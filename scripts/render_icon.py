# /// script
# requires-python = ">=3.10"
# dependencies = [
#     "pillow>=10.0.0",
#     "numpy",
# ]
# ///
"""
render_icon.py: 为 Antigravity-Hans 生成专业现代 Windows 桌面图标与系统托盘图标。
参考 glmquotawatch-gui 的图标制作流水线：
1. 提取 Antigravity 官方彩虹引力弧核心几何视觉（纯透明透底）
2. 生成 4 种现代设计变体（暗黑圆角 Squircle、暗黑+汉化徽章、官方原版白底 Squircle、纯透底浮动弧）
3. 输出 1024x1024 超清 PNG 与标准多分辨率 Windows ICO (16/24/32/48/64/128/256)
4. 输出 Fyne 窗口图标 (gui_assets/appicon.png) 与托盘图标 (gui_assets/tray.png)
5. 自动调用 rsrc 更新 rsrc_windows_amd64.syso
"""

import os
import shutil
import subprocess
import sys
from pathlib import Path
import numpy as np
from PIL import Image, ImageDraw, ImageFilter, ImageFont

if sys.stdout.encoding and sys.stdout.encoding.lower() not in ("utf-8", "utf8"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace", line_buffering=True)
if sys.stderr.encoding and sys.stderr.encoding.lower() not in ("utf-8", "utf8"):
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")

PROJECT_ROOT = Path(__file__).resolve().parent.parent
GUI_ASSETS = PROJECT_ROOT / "gui_assets"
BRAIN_DIR = Path(r"C:\Users\29580\.gemini\antigravity\brain\56ea2751-1ebc-4891-9e95-3a1ecad7ec00")


def extract_pure_arch() -> Image.Image:
    """从官方 icon.png 中将彩虹引力弧完美从白底抠出，返回纯透明透底 RGBA。"""
    src_candidates = [
        PROJECT_ROOT / "icon.png",
        GUI_ASSETS / "antigravity_official_512.png",
        BRAIN_DIR / "antigravity_official_512.png",
    ]
    im = None
    for c in src_candidates:
        if c.exists():
            im = Image.open(c).convert("RGBA")
            break
    if im is None:
        raise FileNotFoundError("未找到 Antigravity 官方 icon.png")

    arr = np.array(im, dtype=np.float32) / 255.0
    orig_alpha = arr[:, :, 3]

    # 彩虹弧在纯白背景 (#FFFFFF) 上，颜色与纯白差异值即前景色透明度
    inv_rgb = 1.0 - arr[:, :, :3]
    alpha_est = np.max(inv_rgb, axis=-1)

    # 抑制白底噪声并平滑抗锯齿边缘
    alpha = np.clip((alpha_est - 0.015) / 0.985, 0.0, 1.0) * (orig_alpha > 0.5)

    # 还原无白底预乘的原始高饱和度 RGB
    fg_rgb = np.zeros_like(arr[:, :, :3])
    mask = alpha > 0.001
    for c in range(3):
        fg_rgb[:, :, c][mask] = 1.0 - inv_rgb[:, :, c][mask] / np.maximum(alpha[mask], 0.05)
    fg_rgb = np.clip(fg_rgb, 0.0, 1.0)

    out = np.zeros_like(arr)
    out[:, :, :3] = fg_rgb * 255.0
    out[:, :, 3] = alpha * 255.0

    im_out = Image.fromarray(out.astype(np.uint8), mode="RGBA")
    bbox = im_out.getbbox()
    if bbox:
        return im_out.crop(bbox)
    return im_out


def create_squircle_mask(dim: int, pad: int, radius: int) -> Image.Image:
    mask = Image.new("L", (dim, dim), 0)
    draw = ImageDraw.Draw(mask)
    draw.rounded_rectangle([pad, pad, dim - pad, dim - pad], radius=radius, fill=255)
    return mask


def render_variant(style: str, pure_arch: Image.Image, size: int = 1024) -> Image.Image:
    scale = 2  # 2x 超采样 -> 2048x2048 后降采样回 1024x1024
    dim = size * scale
    pad = int(80 * scale)
    r_corner = int(220 * scale)

    canvas = Image.new("RGBA", (dim, dim), (0, 0, 0, 0))

    if style in ("dark_squircle", "dark_squircle_badge"):
        # 1. 暗黑底板渐变背景 (#161822 -> #0E1017)
        top_color = np.array([24, 27, 36, 255], dtype=np.float32)
        bot_color = np.array([12, 14, 20, 255], dtype=np.float32)

        bg_arr = np.zeros((dim, dim, 4), dtype=np.uint8)
        for i in range(dim):
            t = i / float(dim)
            col = (1.0 - t) * top_color + t * bot_color
            bg_arr[i, :, :] = col.astype(np.uint8)
        bg_grad = Image.fromarray(bg_arr, mode="RGBA")

        mask = create_squircle_mask(dim, pad, r_corner)
        canvas.paste(bg_grad, (0, 0), mask)

        # 边框微高光与内阴影线
        draw = ImageDraw.Draw(canvas)
        rect_box = [pad, pad, dim - pad, dim - pad]
        draw.rounded_rectangle(
            rect_box,
            radius=r_corner,
            outline=(60, 72, 94, 220),
            width=int(14 * scale),
        )

        # 2. 放置官方彩虹引力弧
        target_w = int(600 * scale)
        arch_w, arch_h = pure_arch.size
        target_h = int(arch_h * (target_w / arch_w))
        resized_arch = pure_arch.resize((target_w, target_h), Image.Resampling.LANCZOS)

        pos_x = (dim - target_w) // 2
        pos_y = (dim - target_h) // 2 - int(10 * scale)

        # 柔和底影与环境光
        shadow = Image.new("RGBA", (dim, dim), (0, 0, 0, 0))
        shadow.paste(resized_arch, (pos_x, pos_y + int(14 * scale)), resized_arch)
        s_arr = np.array(shadow)
        s_arr[:, :, 0] = 0
        s_arr[:, :, 1] = 0
        s_arr[:, :, 2] = 0
        s_arr[:, :, 3] = (s_arr[:, :, 3].astype(np.float32) * 0.6).astype(np.uint8)
        shadow_im = Image.fromarray(s_arr, mode="RGBA").filter(ImageFilter.GaussianBlur(int(20 * scale)))
        canvas = Image.alpha_composite(canvas, shadow_im)

        # 贴上引力弧本体
        canvas.paste(resized_arch, (pos_x, pos_y), resized_arch)

        # 3. 汉化徽章（仅 dark_squircle_badge 变体）
        if style == "dark_squircle_badge":
            badge_r = int(145 * scale)
            badge_cx = dim - pad - int(120 * scale)
            badge_cy = dim - pad - int(120 * scale)
            badge_box = [badge_cx - badge_r, badge_cy - badge_r, badge_cx + badge_r, badge_cy + badge_r]

            # 徽章外阴影
            b_shadow = Image.new("RGBA", (dim, dim), (0, 0, 0, 0))
            b_draw_s = ImageDraw.Draw(b_shadow)
            b_draw_s.ellipse([b - int(4 * scale) for b in badge_box[:2]] + [b + int(4 * scale) for b in badge_box[2:]], fill=(0, 0, 0, 180))
            b_shadow = b_shadow.filter(ImageFilter.GaussianBlur(int(12 * scale)))
            canvas = Image.alpha_composite(canvas, b_shadow)

            # 徽章底板：Google 科技蓝 (#1A73E8) + 纯白高对比边框
            b_draw = ImageDraw.Draw(canvas)
            b_draw.ellipse(badge_box, fill=(26, 115, 232, 255), outline=(255, 255, 255, 230), width=int(9 * scale))

            # 渲染汉字
            font_path = r"C:\Windows\Fonts\msyhbd.ttc"
            if not os.path.exists(font_path):
                font_path = r"C:\Windows\Fonts\msyh.ttc"
            font_size = int(160 * scale)
            font = None
            if os.path.exists(font_path):
                try:
                    font = ImageFont.truetype(font_path, font_size)
                except Exception:
                    font = None
            if font is None:
                font = ImageFont.load_default()

            text = "汉"
            tb = b_draw.textbbox((0, 0), text, font=font)
            tw = tb[2] - tb[0]
            th = tb[3] - tb[1]
            tx = badge_cx - tw // 2 - tb[0]
            ty = badge_cy - th // 2 - tb[1]
            b_draw.text((tx, ty), text, fill=(255, 255, 255, 255), font=font)

    elif style == "official_white_squircle":
        # 官方白底圆角方块（附带极细微灰边框与浅底投影）
        mask = create_squircle_mask(dim, pad, r_corner)
        white_bg = Image.new("RGBA", (dim, dim), (255, 255, 255, 255))
        canvas.paste(white_bg, (0, 0), mask)

        draw = ImageDraw.Draw(canvas)
        rect_box = [pad, pad, dim - pad, dim - pad]
        draw.rounded_rectangle(
            rect_box,
            radius=r_corner,
            outline=(220, 225, 235, 255),
            width=int(10 * scale),
        )

        target_w = int(620 * scale)
        arch_w, arch_h = pure_arch.size
        target_h = int(arch_h * (target_w / arch_w))
        resized_arch = pure_arch.resize((target_w, target_h), Image.Resampling.LANCZOS)
        pos_x = (dim - target_w) // 2
        pos_y = (dim - target_h) // 2 - int(8 * scale)
        canvas.paste(resized_arch, (pos_x, pos_y), resized_arch)

    elif style == "pure_floating":
        # 纯透底无底板彩虹弧
        target_w = int(880 * scale)
        arch_w, arch_h = pure_arch.size
        target_h = int(arch_h * (target_w / arch_w))
        resized_arch = pure_arch.resize((target_w, target_h), Image.Resampling.LANCZOS)

        pos_x = (dim - target_w) // 2
        pos_y = (dim - target_h) // 2

        # 柔和底影
        shadow = Image.new("RGBA", (dim, dim), (0, 0, 0, 0))
        shadow.paste(resized_arch, (pos_x, pos_y + int(18 * scale)), resized_arch)
        s_arr = np.array(shadow)
        s_arr[:, :, 0] = 0
        s_arr[:, :, 1] = 0
        s_arr[:, :, 2] = 0
        s_arr[:, :, 3] = (s_arr[:, :, 3].astype(np.float32) * 0.45).astype(np.uint8)
        shadow_im = Image.fromarray(s_arr, mode="RGBA").filter(ImageFilter.GaussianBlur(int(26 * scale)))
        canvas = Image.alpha_composite(canvas, shadow_im)
        canvas.paste(resized_arch, (pos_x, pos_y), resized_arch)

    return canvas.resize((size, size), Image.Resampling.LANCZOS)


def save_ico(im: Image.Image, ico_path: Path):
    sizes = [(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]
    frames = [im.resize(s, Image.Resampling.LANCZOS) for s in sizes]
    frames[-1].save(
        str(ico_path),
        format="ICO",
        append_images=frames[:-1],
        sizes=sizes,
    )


def generate_tray_preview(im: Image.Image, out_path: Path):
    """生成深浅色双背景下的托盘各尺寸清晰度测试网格。"""
    w, h = 800, 260
    preview = Image.new("RGBA", (w, h), (0, 0, 0, 0))

    draw = ImageDraw.Draw(preview)
    draw.rectangle([0, 0, w, h // 2], fill=(31, 35, 43, 255))
    draw.rectangle([0, h // 2, w, h], fill=(241, 243, 249, 255))
    draw.text((20, 10), "Dark Taskbar Preview (16px, 24px, 32px, 48px, 64px, 128px)", fill=(180, 190, 205, 255))
    draw.text((20, h // 2 + 10), "Light Taskbar Preview", fill=(60, 70, 85, 255))

    sizes = [16, 24, 32, 48, 64, 128]
    cur_x = 30
    for sz in sizes:
        resized = im.resize((sz, sz), Image.Resampling.LANCZOS)
        y1 = 40 + (h // 2 - 40 - sz) // 2
        preview.paste(resized, (cur_x, y1), resized)
        y2 = h // 2 + 35 + (h // 2 - 35 - sz) // 2
        preview.paste(resized, (cur_x, y2), resized)
        cur_x += sz + 30

    preview.save(str(out_path), "PNG")


def main():
    import argparse
    parser = argparse.ArgumentParser(description="Antigravity-Hans 图标生成与渲染流水线")
    parser.add_argument(
        "--variant",
        type=int,
        default=2,
        choices=[1, 2, 3, 4],
        help="选择生效的图标方案：1=暗黑极客 2=暗黑+汉化徽章(默认) 3=官方白底 4=纯透底浮动",
    )
    args = parser.parse_args()

    print("=== [1/5] 读取并提取 Antigravity 官方纯透明彩虹引力弧 ===")
    arch = extract_pure_arch()
    print(f"提取后纯引力弧尺寸: {arch.size}")

    GUI_ASSETS.mkdir(parents=True, exist_ok=True)
    BRAIN_DIR.mkdir(parents=True, exist_ok=True)

    arch.save(str(GUI_ASSETS / "antigravity_pure_arch.png"), "PNG")
    arch.save(str(BRAIN_DIR / "antigravity_pure_arch.png"), "PNG")

    variants = [
        ("variant_1_dark_squircle", "暗黑圆角底板 (推荐·现代极客)", "dark_squircle"),
        ("variant_2_dark_squircle_badge", "暗黑圆角+汉化徽章 (推荐·直观区分)", "dark_squircle_badge"),
        ("variant_3_official_white_squircle", "官方原版白底圆角 (原生标准)", "official_white_squircle"),
        ("variant_4_pure_floating", "纯透底彩虹弧 (轻盈无底板)", "pure_floating"),
    ]

    print("=== [2/5] 渲染 4 种设计变体 (1024x1024 超高清) ===")
    rendered_images = {}
    for key, label, style in variants:
        im = render_variant(style, arch, 1024)
        rendered_images[key] = im
        p_brain = BRAIN_DIR / f"{key}.png"
        im.save(str(p_brain), "PNG")
        print(f"  ✓ {label} -> {key}.png")

    chosen_key = variants[args.variant - 1][0]
    chosen_label = variants[args.variant - 1][1]
    primary_im = rendered_images[chosen_key]
    print(f"[提示] 当前选中生效方案：方案 {args.variant} - {chosen_label}")

    print("=== [3/5] 生成 Windows 多尺寸 ICO 与托盘/窗口资源 ===")
    save_ico(primary_im, PROJECT_ROOT / "icon.ico")
    save_ico(primary_im, BRAIN_DIR / "antigravity_hans.ico")
    print(f"  ✓ 生成标准多分辨率 icon.ico (16-256px)")

    # 窗口大图标
    appicon_png = primary_im.resize((256, 256), Image.Resampling.LANCZOS)
    appicon_png.save(str(GUI_ASSETS / "appicon.png"), "PNG")
    print(f"  ✓ 生成 Fyne 窗口图标 gui_assets/appicon.png (256x256)")

    # 托盘图标
    tray_png = primary_im.resize((64, 64), Image.Resampling.LANCZOS)
    tray_png.save(str(GUI_ASSETS / "tray.png"), "PNG")
    print(f"  ✓ 生成 Fyne 托盘图标 gui_assets/tray.png (64x64)")

    # 托盘清晰度测试图（保存至 brain 供审阅）
    generate_tray_preview(primary_im, BRAIN_DIR / "tray_legibility_test.png")
    print(f"  ✓ 生成清晰度预览图 brain/tray_legibility_test.png")

    print("=== [4/5] 编译 Windows 资源对象 rsrc_windows_amd64.syso ===")
    rsrc = shutil.which("rsrc") or str(Path.home() / "go" / "bin" / "rsrc.exe")
    if Path(rsrc).exists():
        syso_path = PROJECT_ROOT / "rsrc_windows_amd64.syso"
        ico_path = PROJECT_ROOT / "icon.ico"
        subprocess.run([rsrc, "-ico", str(ico_path), "-o", str(syso_path)], cwd=str(PROJECT_ROOT), check=True)
        print(f"  ✓ 成功编译 {syso_path.name} (大小: {syso_path.stat().st_size} bytes)")
    else:
        print("  [警告] 未找到 rsrc，跳过 syso 生成")

    print("=== [5/5] 完成图标资源更新 ===")


if __name__ == "__main__":
    main()
