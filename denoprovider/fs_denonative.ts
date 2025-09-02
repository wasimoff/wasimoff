import { ProviderStorageFileSystem } from "@wasimoff/storage/index.ts";
import { join } from "jsr:@std/path@1";

const logprefix = ["%c[DenoFileSystem]", "color: purple;"];

export class DenoFileSystem implements ProviderStorageFileSystem {
  // base directory on disk
  public readonly path: string;

  // open a directory as persistent backing
  private constructor(path: string) {
    this.path = path;
  }

  // make sure the directory exists and return storage
  static async open(directory: string) {
    // create the directory
    await Deno.mkdir(directory, { recursive: true, mode: 0o700 });
    return new DenoFileSystem(directory);
  }

  // list all files in directory
  async list(): Promise<string[]> {
    const list = [];
    for await (const e of Deno.readDir(this.path)) {
      if (e.isFile) list.push(e.name);
    }
    console.debug(...logprefix, `has ${list.length} files:`, list);
    return list;
  }

  // return file from directory
  async get(filename: string): Promise<File | undefined> {
    try {
      const buf = await Deno.readFile(join(this.path, filename));
      return new File([buf], filename);
    } catch (err: unknown) {
      if (err instanceof Deno.errors.NotFound) return undefined;
      else throw err;
    }
  }

  // check if file is in directory
  async has(filename: string): Promise<boolean> {
    try {
      const info = await Deno.stat(join(this.path, filename));
      return info.isFile;
    } catch (err: unknown) {
      if (err instanceof Deno.errors.NotFound) return false;
      else throw err;
    }
  }

  // store a new file in directory
  async put(filename: string, file: File): Promise<File> {
    console.debug(...logprefix, `store:`, file);
    await Deno.writeFile(join(this.path, filename), await file.bytes());
    return file;
  }

  // remove a file from directory
  async delete(filename: string): Promise<boolean> {
    console.debug(...logprefix, `delete:`, filename);
    try {
      await Deno.remove(join(this.path, filename));
      return true;
    } catch (err: unknown) {
      if (err instanceof Deno.errors.NotFound) return false;
      else throw err;
    }
  }
}
