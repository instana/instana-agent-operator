"""Make public archives byte-stable when re-running the same promotion."""

import gzip
import io
from pathlib import Path
import sys
import tarfile
import zipfile


def normalize(tgz, zip_path):
    raw = io.BytesIO()
    with tarfile.open(tgz, "r:gz") as src, tarfile.open(fileobj=raw, mode="w", format=tarfile.USTAR_FORMAT) as dst:
        for member in sorted(src.getmembers(), key=lambda m: m.name):
            member.mtime = member.uid = member.gid = 0
            member.uname = member.gname = ""
            member.pax_headers = {}
            dst.addfile(member, src.extractfile(member) if member.isfile() else None)
    Path(tgz).write_bytes(gzip.compress(raw.getvalue(), mtime=0))
    with zipfile.ZipFile(zip_path) as src:
        files = {name: src.read(name) for name in src.namelist() if not name.endswith("/")}
    with zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_DEFLATED) as dst:
        for name, data in sorted(files.items()):
            info = zipfile.ZipInfo(name, date_time=(1980, 1, 1, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = 0o100644 << 16
            dst.writestr(info, data)


if __name__ == "__main__":
    normalize(*sys.argv[1:])
