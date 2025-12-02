#!/usr/bin/env python3
# -*- coding: utf-8 -*-

from pathlib import Path
from datetime import datetime
import json
import qrcode
import os
import platform

from PIL import Image

import tkinter as tk
from tkinter import filedialog, messagebox
from tkinter import ttk

# ---------- PATH ----------

BASE_DIR = Path(r"C:\Users\User\Downloads\Open House TSE-20251109T161746Z-1-001\Open House TSE")

FRAME_FILES = {
    "Black": BASE_DIR / "BF.png",
    "Brown": BASE_DIR / "FT.png",
    "Red":   BASE_DIR / "RF.png",
}

INPUT_DIR  = BASE_DIR / r"input_images\2025_11_09"
OUTPUT_DIR = BASE_DIR / "output_images"
STATE_FILE = BASE_DIR / "last_selection.json"

CLIENT_SECRETS   = BASE_DIR / "client_secrets.json"
CREDENTIALS_PATH = BASE_DIR / "credentials.json"
ENABLE_GDRIVE_UPLOAD = False

# ---------- PARAMS ----------

FRAME_W = 1200
FRAME_H = 1800
SLOT_W  = 542
SLOT_H  = 408

SLOTS_PX = [
    (45, 80),
    (45, 490),
    (45, 900),
    (630, 80),
    (630, 490),
    (630, 900),
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

def load_last_selection():
    if STATE_FILE.exists():
        try:
            data = json.loads(STATE_FILE.read_text(encoding="utf-8"))
            return [Path(p) for p in data.get("slots", [])]
        except Exception:
            pass
    return [Path()] * 3

def save_last_selection(paths):
    data = {"slots": [str(p) for p in paths]}
    STATE_FILE.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")

def auto_pick_from_input_dir() -> Path:
    exts = {".jpg", ".jpeg", ".png", ".webp", ".bmp"}
    cand = [p for p in INPUT_DIR.glob("*") if p.suffix.lower() in exts]
    if not cand:
        raise FileNotFoundError(f"ไม่พบภาพใน {INPUT_DIR}")
    cand.sort(key=lambda p: p.stat().st_mtime)
    return cand[-1]

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

# ---------- Core: Generate Frame ----------

def generate_frame_image(frame_path: Path, selected_paths):
    ensure_dirs()

    frame = Image.open(frame_path).convert("RGBA")
    W, H = frame.size

    canvas = Image.new("RGBA", (W, H), (255, 255, 255, 0))

    for i, (slot_x, slot_y) in enumerate(SLOTS_PX):
        img_index = i % len(selected_paths)
        src = selected_paths[img_index]
        im = Image.open(src).convert("RGBA")
        im = fit_inside(im, SLOT_W, SLOT_H)
        paste_x = slot_x + (SLOT_W - im.width) // 2
        paste_y = slot_y + (SLOT_H - im.height) // 2
        canvas.paste(im, (paste_x, paste_y), im)

    canvas = Image.alpha_composite(canvas, frame)

    out = unique_output_path(OUTPUT_DIR)
    canvas.save(out)
    print(f"🎉 บันทึกแล้ว: {out}")

    qr_path = None
    if ENABLE_GDRIVE_UPLOAD:
        url = upload_to_gdrive(out)
        if url:
            qr_path = OUTPUT_DIR / f"{out.stem}_qr.png"
            make_qr_from_url(url, qr_path)
            print("📱 สแกน QR ได้เลย")

    return out, qr_path

# ---------- GUI Class ----------

class FrameApp:

    def __init__(self, root: tk.Tk):
        self.root = root
        self.root.title("Open House Frame Generator")
        self.root.configure(bg="#f3f4f6")
        self.root.minsize(640, 420)
        self._center_window(720, 480)

        self._setup_style()

        last = load_last_selection()
        self.selected_paths = []
        for p in last[:3]:
            self.selected_paths.append(p if p and p.exists() else None)

        self.slot_labels_var = []
        for p in self.selected_paths:
            text = p.name if (p and p.exists()) else "(ยังไม่เลือก)"
            self.slot_labels_var.append(tk.StringVar(value=text))

        self.frame_var = tk.StringVar(value="Black")
        self.status_var = tk.StringVar(value="พร้อมทำงาน")

        self._build_ui()

    # -------- UI Style & Layout --------

    def _setup_style(self):
        style = ttk.Style()
        try:
            style.theme_use("clam")
        except Exception:
            pass

        style.configure("Header.TLabel",
                        font=("Segoe UI", 18, "bold"),
                        foreground="#111827",
                        background="#f3f4f6")

        style.configure("SubHeader.TLabel",
                        font=("Segoe UI", 10),
                        foreground="#4b5563",
                        background="#f3f4f6")

        style.configure("TLabel",
                        font=("Segoe UI", 10),
                        background="#f9fafb")

        style.configure("TButton",
                        font=("Segoe UI", 10, "bold"),
                        padding=(10, 5))

        style.configure("Accent.TButton",
                        font=("Segoe UI", 11, "bold"),
                        padding=(12, 8))

        style.configure("Card.TLabelframe",
                        background="#f9fafb")
        style.configure("Card.TLabelframe.Label",
                        font=("Segoe UI", 11, "bold"),
                        foreground="#111827",
                        background="#f9fafb")

    def _center_window(self, w, h):
        sw = self.root.winfo_screenwidth()
        sh = self.root.winfo_screenheight()
        x = int((sw - w) / 2)
        y = int((sh - h) / 3)
        self.root.geometry(f"{w}x{h}+{x}+{y}")

    def _build_ui(self):

        main = ttk.Frame(self.root, padding=16)
        main.pack(fill="both", expand=True)

        header_frame = ttk.Frame(main)
        header_frame.pack(fill="x", pady=(0, 10))

        ttk.Label(header_frame,
                  text="Open House Frame Generator",
                  style="Header.TLabel").pack(anchor="w")

        ttk.Label(header_frame,
                  text="เลือกรูป 3 รูป + เลือกกรอบ แล้วกด Generate ได้เลย ✨",
                  style="SubHeader.TLabel").pack(anchor="w", pady=(2, 0))

        ttk.Separator(main, orient="horizontal").pack(fill="x", pady=8)

        content = ttk.Frame(main)
        content.pack(fill="both", expand=True)

        # เลือกกรอบ
        frame_style = ttk.Labelframe(content, text=" เลือกกรอบ ",
                                     style="Card.TLabelframe")
        frame_style.pack(side="left", fill="both", expand=True,
                         padx=(0, 8), pady=4, ipadx=6, ipady=4)

        for name in FRAME_FILES.keys():
            ttk.Radiobutton(frame_style, text=name,
                            value=name, variable=self.frame_var).pack(anchor="w", padx=6, pady=4)

        # เลือกรูป
        frame_slots = ttk.Labelframe(content, text=" เลือกรูปภาพ 3 รูป ",
                                     style="Card.TLabelframe")
        frame_slots.pack(side="right", fill="both", expand=True,
                         padx=(8, 0), pady=4, ipadx=6, ipady=4)

        for i in range(3):
            row = ttk.Frame(frame_slots)
            row.pack(fill="x", pady=4, padx=4)

            ttk.Label(row, text=f"ช่อง {i+1}:", width=8).pack(side="left")
            ttk.Label(row, textvariable=self.slot_labels_var[i], width=32).pack(side="left", padx=4)

            ttk.Button(row, text="เลือกไฟล์",
                       command=lambda idx=i: self.choose_file(idx)).pack(side="right")

        # ปุ่มล่าง
        buttons_frame = ttk.Frame(main)
        buttons_frame.pack(fill="x", pady=(12, 4))

        ttk.Button(buttons_frame, text="✨ Generate + Upload",
                   style="Accent.TButton", command=self.on_generate).pack(side="left")

        ttk.Button(buttons_frame, text="📁 เปิดโฟลเดอร์ Output",
                   command=self.open_output_folder).pack(side="left", padx=8)

        # Status bar
        status_frame = ttk.Frame(main)
        status_frame.pack(fill="x", pady=(8, 0))

        ttk.Label(status_frame, textvariable=self.status_var,
                  anchor="w").pack(fill="x")

    # -------- GUI Actions --------

    def choose_file(self, idx: int):
        init_dir = INPUT_DIR if INPUT_DIR.exists() else BASE_DIR
        filetypes = [("Image files", "*.jpg *.jpeg *.png *.webp *.bmp")]
        path_str = filedialog.askopenfilename(initialdir=str(init_dir), filetypes=filetypes)
        if not path_str:
            return
        p = Path(path_str)
        if not p.exists():
            messagebox.showerror("Error", "ไฟล์นี้ไม่พบในระบบ")
            return
        self.selected_paths[idx] = p
        self.slot_labels_var[idx].set(p.name)

    def open_output_folder(self):
        try:
            path = OUTPUT_DIR
            if not path.exists():
                OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

            if platform.system() == "Windows":
                os.startfile(str(path))
            elif platform.system() == "Darwin":
                import subprocess
                subprocess.Popen(["open", str(path)])
            else:
                import subprocess
                subprocess.Popen(["xdg-open", str(path)])

            self.status_var.set("เปิดโฟลเดอร์ Output แล้ว")
        except Exception as e:
            messagebox.showerror("Error", f"เปิดโฟลเดอร์ไม่สำเร็จ: {e}")
            self.status_var.set("เปิดโฟลเดอร์ไม่สำเร็จ")

    def on_generate(self):
        frame_name = self.frame_var.get()
        frame_path = FRAME_FILES.get(frame_name)

        if not frame_path or not frame_path.exists():
            messagebox.showerror("Error", f"ไม่พบไฟล์กรอบ {frame_name}")
            return

        selected = []
        try:
            for i in range(3):
                p = self.selected_paths[i]
                if p is None or not p.exists():
                    p = auto_pick_from_input_dir()
                    self.selected_paths[i] = p
                    self.slot_labels_var[i].set(p.name)
                selected.append(p)
        except FileNotFoundError as e:
            messagebox.showerror("Error", str(e))
            return

        save_last_selection(selected)

        self.status_var.set("กำลังสร้างภาพและอัปโหลด...")
        self.root.update_idletasks()

        try:
            out_path, qr_path = generate_frame_image(frame_path, selected)
        except Exception as e:
            messagebox.showerror("Error", f"เกิดข้อผิดพลาด: {e}")
            self.status_var.set("เกิดข้อผิดพลาด")
            return

        msg = f"บันทึกไฟล์ภาพที่:\n{out_path}"
        if qr_path:
            msg += f"\n\nบันทึก QR ที่:\n{qr_path}"

        messagebox.showinfo("สำเร็จ!", msg)
        self.status_var.set("เสร็จสิ้น ✅")

# ---------- main ----------

def main():
    root = tk.Tk()
    app = FrameApp(root)
    root.mainloop()

if __name__ == "__main__":
    main()
