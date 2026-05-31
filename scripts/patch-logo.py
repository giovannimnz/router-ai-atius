#!/usr/bin/env python3
"""Patch embedded assets in new-api binary with Atius branding."""
import sys
import os

def patch_png(binary_path, new_png_path):
    """Patch embedded PNG (logo.png) in binary."""
    with open(binary_path, 'rb') as f:
        binary = f.read()

    with open(new_png_path, 'rb') as f:
        logo_data = f.read()

    png_sig = b'\x89PNG\r\n\x1a\n'
    pos = binary.find(png_sig)
    if pos < 0:
        print(f"PNG signature not found in binary", file=sys.stderr)
        return False

    iend_pos = binary.find(b'IEND', pos)
    if iend_pos < 0:
        print(f"IEND chunk not found", file=sys.stderr)
        return False

    end_of_png = iend_pos + 12
    new_binary = binary[:pos] + logo_data + binary[end_of_png:]

    with open(binary_path, 'wb') as f:
        f.write(new_binary)

    print(f"Patched logo.png ({len(logo_data)} bytes)")
    return True

def patch_ico(binary_path, new_ico_path):
    """Patch embedded ICO (favicon.ico) in binary."""
    with open(binary_path, 'rb') as f:
        binary = f.read()

    with open(new_ico_path, 'rb') as f:
        ico_data = f.read()

    ico_sig = b'\x00\x00\x01\x00'
    pos = binary.find(ico_sig)
    if pos < 0:
        print(f"ICO signature not found in binary (this is OK if no ico embedded)")
        return True

    # Find next embedded file marker after ICO (or EOF)
    # ICO files end with the last image data - we look for another known signature
    # For simplicity, replace from ico_sig to next PNG (if any) or to EOF
    next_png = binary.find(b'\x89PNG\r\n\x1a\n', pos + 1)
    if next_png > 0:
        end_pos = next_png
    else:
        end_pos = len(binary)

    new_binary = binary[:pos] + ico_data + binary[end_pos:]
    with open(binary_path, 'wb') as f:
        f.write(new_binary)

    print(f"Patched favicon.ico ({len(ico_data)} bytes)")
    return True

def patch_svg(binary_path, new_svg_path):
    """Patch embedded SVG (logo.svg) in binary."""
    with open(binary_path, 'rb') as f:
        binary = f.read()

    with open(new_svg_path, 'rb') as f:
        svg_data = f.read()

    svg_sig = b'<svg'
    pos = binary.find(svg_sig)
    if pos < 0:
        print(f"SVG signature not found in binary", file=sys.stderr)
        return False

    end_pos = binary.find(b'</svg>', pos)
    if end_pos < 0:
        print(f"</svg> not found", file=sys.stderr)
        return False
    end_pos += 6  # include </svg>

    new_binary = binary[:pos] + svg_data + binary[end_pos:]
    with open(binary_path, 'wb') as f:
        f.write(new_binary)

    print(f"Patched logo.svg ({len(svg_data)} bytes)")
    return True

def patch_index_html(binary_path, new_title):
    """Patch title in embedded index.html."""
    with open(binary_path, 'rb') as f:
        binary = f.read()

    html_sig = b'<!doctype html>'
    pos = binary.lower().find(html_sig)
    if pos < 0:
        print(f"HTML signature not found in binary", file=sys.stderr)
        return True

    title_start = binary.find(b'<title>', pos)
    if title_start < 0:
        print(f"<title> not found", file=sys.stderr)
        return True

    title_end = binary.find(b'</title>', title_start)
    if title_end < 0:
        print(f"</title> not found", file=sys.stderr)
        return True

    title_content = binary[title_start + 7:title_end]
    new_title_tag = f'<title>{new_title}</title>'.encode()
    new_binary = binary[:title_start] + new_title_tag + binary[title_end + 8:]

    with open(binary_path, 'wb') as f:
        f.write(new_binary)

    print(f"Patched title: {title_content.decode()} -> {new_title}")
    return True

def main():
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} <binary> <logo.png> [logo.svg] [favicon.ico]")
        sys.exit(1)

    binary = sys.argv[1]
    png_path = sys.argv[2]
    svg_path = sys.argv[3] if len(sys.argv) > 3 else None
    ico_path = sys.argv[4] if len(sys.argv) > 4 else None

    if not os.path.exists(binary):
        print(f"Binary not found: {binary}")
        sys.exit(1)

    print(f"Patching {binary}...")

    ok = True
    if os.path.exists(png_path):
        ok &= patch_png(binary, png_path)
    if svg_path and os.path.exists(svg_path):
        ok &= patch_svg(binary, svg_path)
    if ico_path and os.path.exists(ico_path):
        ok &= patch_ico(binary, ico_path)

    # Always patch index.html title
    ok &= patch_index_html(binary, "Atius Router")

    sys.exit(0 if ok else 1)

if __name__ == '__main__':
    main()
