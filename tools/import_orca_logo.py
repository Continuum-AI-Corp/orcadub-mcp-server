#!/usr/bin/env python3
"""Import the standalone Orca ANSI logo into embedded CLI assets."""

import argparse
import re
from pathlib import Path

ANSI = re.compile(rb"\x1b\[[0-9;]*m")
START = b"cat <<'ORCA_EOF'\n"
END = b"\nORCA_EOF\n"


def extract_payload(source: bytes) -> bytes:
    if START not in source or END not in source:
        raise ValueError("source does not contain the ORCA_EOF heredoc")
    payload = source.split(START, 1)[1].split(END, 1)[0]
    lines = payload.splitlines()
    if len(lines) != 20:
        raise ValueError(f"logo has {len(lines)} rows, want 20")
    for index, line in enumerate(lines):
        visible = ANSI.sub(b"", line).decode("utf-8")
        if len(visible) != 40:
            raise ValueError(
                f"logo row {index} has width {len(visible)}, want 40"
            )
    return payload + b"\n"


def plain_payload(color_payload: bytes) -> bytes:
    rows = []
    for line in color_payload.rstrip(b"\n").splitlines():
        visible = ANSI.sub(b"", line).decode("utf-8")
        rows.append(visible.translate(str.maketrans({"▀": "█", "▄": "█"})))
    return ("\n".join(rows) + "\n").encode("utf-8")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("source", type=Path)
    parser.add_argument("output_dir", type=Path)
    args = parser.parse_args()

    color = extract_payload(args.source.read_bytes())
    plain = plain_payload(color)
    args.output_dir.mkdir(parents=True, exist_ok=True)
    (args.output_dir / "orca_logo_color.ansi").write_bytes(color)
    (args.output_dir / "orca_logo_plain.txt").write_bytes(plain)


if __name__ == "__main__":
    main()
