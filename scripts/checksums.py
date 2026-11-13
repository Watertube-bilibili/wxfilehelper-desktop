import hashlib
from pathlib import Path

dist = Path(__file__).resolve().parents[1] / "dist"
artifacts = sorted([*dist.glob("*.exe"), *dist.glob("*.zip"), *dist.glob("*.dmg")])
lines = []
for path in artifacts:
    with path.open("rb") as stream:
        digest = hashlib.file_digest(stream, "sha256").hexdigest()
    lines.append(f"{digest}  {path.name}")
(dist / "SHA256SUMS.txt").write_text("\n".join(lines) + "\n", encoding="utf-8")
print("\n".join(lines))
