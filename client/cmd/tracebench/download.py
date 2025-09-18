#!/usr/bin/env python3
# /// script
# dependencies = [ "requests", "pandas", "tqdm" ]
# ///

import sys, os, zipfile
import requests, pandas, tqdm


# download, concatenate and then re-save as single-file compressed CSVs
def download(names: list[str], directory="dataset"):
    os.makedirs(directory, exist_ok=True)
    for name in names:
        # construct filenames
        archive = os.path.join(directory, f"{name}.zip")
        csv = os.path.join(directory, f"{name}.csv.gz")
        if os.path.exists(csv):
            print(name, "> file exists:", csv)
        else:
            # maybe download the original archive
            if not os.path.exists(archive):
                print(name, "> download", link(name))
                fetch(link(name), archive)
            # concatenate to a single dataframe for easier analysis
            print(name, "> concatenate to a single dataframe")
            concatenate(archive, dest=csv)


# download a file to disk with progress bar
def fetch(url: str, dest: str):
    response = requests.get(url, stream=True)
    response.raise_for_status()
    size = int(response.headers.get("content-length", 0))
    with open(dest, "wb") as w, tqdm.tqdm(
        desc=dest, total=size, unit_scale=True, unit="B"
    ) as bar:
        for data in response.iter_content(chunk_size=32768):
            bar.update(w.write(data))


# concatenate an archive to a single dataframe and save it as .csv.gz
def concatenate(archive: str, dest: str):
    with open(archive, "rb") as z:
        df = dataframe(z)
    print(f"write to {dest} (might take a while) ...")
    df.to_csv(dest, index=False, compression="infer")


# concat all the CSVs inside an archive to pd.DataFrame
def dataframe(archive):
    with zipfile.ZipFile(archive) as z:
        files = [f for f in z.filelist if not f.is_dir()]
        files.sort(key=lambda f: f.filename)
        return pandas.concat(
            [
                pandas.read_csv(z.open(f), dtype=float)
                for f in tqdm.tqdm(files, desc="pandas.concat")
            ],
            ignore_index=True,
        )


# return the full URL for a given zip file
def link(name: str):
    return f"https://sir-dataset.obs.cn-east-3.myhuaweicloud.com/datasets/private_dataset/{name}.zip"


private_dataset = [
    "requests_minute",
    "requests_second",
    "function_delay_minute",
    "function_delay_second",
    "platform_delay_minute",
    "platform_delay_second",
    "cpu_usage_minute",
    "memory_usage_minute",
    "cpu_limit_minute",
    "memory_limit_minute",
    "instances_minute",
]


if __name__ == "__main__":
    files = sys.argv[1:] or ["requests_minute", "function_delay_minute"]
    if not all(n in private_dataset for n in files):
        print(f"usage: {sys.argv[0]} <dataset> [...]", file=sys.stderr)
        print("from:", private_dataset, file=sys.stderr)
        exit(1)
    download(files)
