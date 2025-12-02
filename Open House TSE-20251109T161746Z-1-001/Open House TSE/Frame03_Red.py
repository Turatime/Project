#!/usr/bin/env python3
# -*- coding: utf-8 -*-

from PIL import Image
from pathlib import Path
from datetime import datetime
import qrcode, json

# ---------- PATH ----------

BASE_DIR = Path(r"C:\Users\User\Downloads\Open House TSE-20251109T161746Z-1-001\Open House TSE")  # ✅ แก้ให้ตรงกับของคุณ
FRAME_PATH = BASE_DIR / "RF.png"
INPUT_DIR  = BASE_DIR / r"input_images\2025_11_09"
OUTPUT_DIR = BASE_DIR / "output_images"
STATE_FILE = BASE_DIR / "last_selection.json"

CLIENT_SECRETS   = BASE_DIR / "client_secrets.json"
CREDENTIALS_PATH = BASE_DIR / "credentials.json"
ENABLE_GDRIVE_UPLOAD = True

# ---------- พารามิเตอร์ ----------
FRAME_W = 1200
FRAME_H = 1800
SLOT_W  = 542
SLOT_H  = 408

# พิกัดช่อง (x, y)
SLOTS_PX = [
    (45, 80),    # ซ้ายบน
    (45, 490),    # ซ้ายกลาง
    (45, 900),    # ซ้ายล่าง
    (630, 80),   # ขวาบน
    (630, 490),   # ขวากลาง
    (630, 900),   # ขวาล่าง
]


# ---------- Utility ----------
def fit_inside(im, max_w, max_h):
    w, h = im.size
    scale = min(max_w / w, max_h / h)
    return im.resize((int(w * scale), int(h * scale)), Image.Resampling.LANCZOS)

def unique_output_path(out_dir: Path, prefix="frame6slots", ext=".png") -> Path:
    ts = datetime.now().strftime("%Y%m%d-%H%M%S")
    p = out_dir / f"{prefix}_{ts}{ext}"
    if not p.exists():
        return p
    i = 1
    while True:
        q = out_dir / f"{prefix}_{ts}_{i}{ext}"
        if not q.exists():
            return q
        i += 1

def ensure_dirs():
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

# ---------- เลือกภาพ ----------
def _tk_file_dialog(initial: Path) -> Path | None:
    try:
        import tkinter as tk
        from tkinter import filedialog
        root = tk.Tk()
        root.withdraw()
        filetypes = [("Image files", "*.jpg *.jpeg *.png *.webp *.bmp")]
        path = filedialog.askopenfilename(initialdir=str(initial), filetypes=filetypes)
        if not path:
            return None
        return Path(path)
    except Exception:
        return None

def load_last_selection() -> list[Path]:
    if STATE_FILE.exists():
        try:
            data = json.loads(STATE_FILE.read_text(encoding="utf-8"))
            return [Path(p) for p in data.get("slots", [])]
        except Exception:
            pass
    return [Path()] * 3

def save_last_selection(paths: list[Path]):
    data = {"slots": [str(p) for p in paths]}
    STATE_FILE.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")

def auto_pick_from_input_dir() -> Path:
    exts = {".jpg", ".jpeg", ".png", ".webp", ".bmp"}
    cand = [p for p in INPUT_DIR.glob("*") if p.suffix.lower() in exts]
    if not cand:
        raise FileNotFoundError(f"ไม่พบภาพใน {INPUT_DIR}")
    cand.sort(key=lambda p: p.stat().st_mtime)
    return cand[-1]

def pick_images_for_slots():
    prev = load_last_selection()
    chosen = prev[:]

    for idx in range(3):
        print(f"🖼 เลือกไฟล์สำหรับช่อง {idx+1} (Cancel = ใช้ไฟล์เดิม)")
        picked = _tk_file_dialog(INPUT_DIR if INPUT_DIR.exists() else BASE_DIR)
        if picked and picked.exists():
            chosen[idx] = picked
            print(f"   → ใช้ไฟล์ใหม่: {picked.name}")
        else:
            if chosen[idx] and chosen[idx].exists():
                print(f"   → คงไฟล์เดิม: {chosen[idx].name}")
            else:
                fallback = auto_pick_from_input_dir()
                chosen[idx] = fallback
                print(f"   → ไม่มีไฟล์เดิม เลือกอัตโนมัติ: {fallback.name}")

    save_last_selection(chosen)
    return chosen

# ---------- Google Drive ----------
def _resolve_client_secrets(p: Path) -> Path:
    if p.exists():
        return p
    alt = sorted(BASE_DIR.glob("client_secret_*.json"))
    if alt:
        print(f"ℹ️ ใช้ไฟล์ OAuth: {alt[-1].name}")
        return alt[-1]
    raise FileNotFoundError(f"ไม่พบไฟล์ OAuth: {p}")

def upload_to_gdrive(local_path: Path) -> str:
    if not ENABLE_GDRIVE_UPLOAD:
        return ""
    try:
        from pydrive2.auth import GoogleAuth
        from pydrive2.drive import GoogleDrive
    except ImportError:
        print("⚠️ ติดตั้ง pydrive2 ก่อน: pip install pydrive2")
        return ""
    gauth = GoogleAuth()
    gauth.DEFAULT_SETTINGS['client_config_file'] = str(_resolve_client_secrets(CLIENT_SECRETS))
    gauth.DEFAULT_SETTINGS['oauth_scope'] = ['https://www.googleapis.com/auth/drive.file']
    gauth.DEFAULT_SETTINGS['get_refresh_token'] = True
    gauth.LoadCredentialsFile(str(CREDENTIALS_PATH))
    try:
        if gauth.credentials is None:
            gauth.LocalWebserverAuth()
        elif gauth.access_token_expired:
            gauth.Refresh()
        else:
            gauth.Authorize()
    except Exception:
        gauth.CommandLineAuth()
    gauth.SaveCredentialsFile(str(CREDENTIALS_PATH))
    drive = GoogleDrive(gauth)
    f = drive.CreateFile({'title': Path(local_path).name})
    f.SetContentFile(str(local_path))
    f.Upload()
    f.InsertPermission({'type': 'anyone', 'value': 'anyone', 'role': 'reader'})
    file_id = f['id']
    embed_url = f"https://drive.google.com/uc?export=view&id={file_id}"
    print("🖼 Image (embed):", embed_url)
    return embed_url

def make_qr_from_url(url: str, output_path: Path):
    img = qrcode.make(url)
    img.save(output_path)
    print(f"✅ QR saved to {output_path}")

# ---------- Main ----------
def main():
    ensure_dirs()
    frame = Image.open(FRAME_PATH).convert("RGBA")
    W, H = frame.size

    selected = pick_images_for_slots()

    canvas = Image.new("RGBA", (W, H), (255, 255, 255, 0))

    # วางครบ 6 ช่อง (วนรูป 3 รูปซ้ำ)
    for i, (slot_x, slot_y) in enumerate(SLOTS_PX):
        img_index = i % len(selected)
        src = selected[img_index]
        im = Image.open(src).convert("RGBA")
        im = fit_inside(im, SLOT_W, SLOT_H)
        paste_x = slot_x + (SLOT_W - im.width) // 2
        paste_y = slot_y + (SLOT_H - im.height) // 2
        canvas.paste(im, (paste_x, paste_y), im)

    canvas = Image.alpha_composite(canvas, frame)
    out = unique_output_path(OUTPUT_DIR)
    canvas.save(out)
    print(f"🎉 บันทึกแล้ว: {out}")

    if ENABLE_GDRIVE_UPLOAD:
        url = upload_to_gdrive(out)
        if url:
            qr_path = OUTPUT_DIR / f"{out.stem}_qr.png"
            make_qr_from_url(url, qr_path)
            print("📱 สแกน QR ได้เลย")

if __name__ == "__main__":
    main()
