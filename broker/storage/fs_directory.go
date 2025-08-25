package storage

import (
	"fmt"
	"iter"
	"log"
	"os"
	"path"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/peterbourgon/diskv/v3"
	bolt "go.etcd.io/bbolt"
)

type DirectoryFileStorage struct {
	db *bolt.DB
	kv *diskv.Diskv
}

func NewDirectoryFileStorage(basedir string) *FileStorage {

	// make sure the basedir exists
	if err := os.MkdirAll(basedir, 0755); err != nil {
		log.Fatalf("dirfs: cannot create directory: %s", err)
	}

	// open the boltdb file for aliases
	db, err := bolt.Open(path.Join(basedir, "lookup.db"), 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		// to keep the API clean, we just abort in here since this happens only at startup
		log.Fatalf("dirfs: cannot open lookup db: %s", err)
	}

	// ensure that lookup bucket exists
	err = db.Update(func(tx *bolt.Tx) (err error) {
		_, err = tx.CreateBucketIfNotExists(lookupBucket)
		return
	})
	if err != nil {
		log.Fatalf("dirfs: cannot create bucket: %s", err)
	}

	// open the diskv storage
	kv := diskv.New(diskv.Options{
		// 64 MB cache
		CacheSizeMax: 64 * 1024 * 1024,
		// store files in blobs subdirectory
		BasePath: basedir,
		Transform: func(s string) []string {
			return []string{"blob"}
		},
	})

	return &FileStorage{
		AbstractFileStorage: &DirectoryFileStorage{db, kv},
	}
}

// Insert a new file into the Storage. The optional `name` will be inserted
// into the lookup table and can be used to resolve the file later.
func (fs *DirectoryFileStorage) Insert(name, media string, blob []byte) (file *File, err error) {

	// check the media type first because that's cheapest
	media, err = CheckMediaType(media)
	if err != nil {
		return nil, fmt.Errorf("media: %w", err)
	}

	// use a *File struct to obtain the content hash
	file = NewFile(media, blob)
	ref := file.Ref()

	// store the file on disk
	if err = fs.kv.Write(ref, blob); err != nil {
		return nil, fmt.Errorf("failed to write blob: %w", err)
	}

	// insert lookup name in boltdb bucket
	if name != "" {
		err = fs.db.Update(func(tx *bolt.Tx) error {
			return tx.Bucket(lookupBucket).Put([]byte(name), []byte(ref))
		})
	}

	return file, err

}

// Get a File from Storage, either by Ref or a friendly name in lookup map.
func (fs *DirectoryFileStorage) Get(nameOrRef string) (f *File) {
	// attempt to fetch by ref directly
	f = fs.get(nameOrRef)
	if f == nil {
		// try to lookup a friendly name
		fs.db.View(func(tx *bolt.Tx) error {
			ref := tx.Bucket(lookupBucket).Get([]byte(nameOrRef))
			f = fs.get(string(ref))
			return nil
		})
	}
	// at this point f is either nil or successfully resolved ...
	return
}

func (fs *DirectoryFileStorage) get(ref string) (f *File) {

	// get file from disk
	blob, err := fs.kv.Read(ref)
	if blob == nil || err != nil {
		return nil // no such file?
	}

	// parse the mediatype
	media := mimetype.Detect(blob).String()

	// return as *File
	// TODO: do we need to copy the blob slice?
	return NewFile(media, blob)

}

// Iterator over all Files in the storage.
func (fs *DirectoryFileStorage) All() iter.Seq2[string, *File] {
	return func(yield func(string, *File) bool) {
		cancel := make(chan struct{})
		for key := range fs.kv.Keys(cancel) {
			file := fs.get(key)
			if file == nil {
				panic("dirfs: got a nil *File while iterating in All()")
			}
			if !yield(key, file) {
				close(cancel)
				return
			}
		}
	}
}
