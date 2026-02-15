from __future__ import annotations

import argparse
from collections import Counter
from pathlib import Path
import shutil
from typing import Sequence

from .converter import ConversionConfig, PdfBookConverter, build_default_ocr_backend


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description=(
            "Convert a PDF into one image per page plus per-page Markdown and LaTeX files."
        )
    )
    parser.add_argument("--input", required=True, type=Path, help="Input PDF path")
    parser.add_argument("--latex-dir", required=True, type=Path, help="Output LaTeX directory")
    parser.add_argument(
        "--markdown-dir", required=True, type=Path, help="Output Markdown directory"
    )
    parser.add_argument("--images-dir", required=True, type=Path, help="Output images directory")
    parser.add_argument("--dpi", type=int, default=220, help="Rasterization DPI (default: 220)")
    parser.add_argument("--start-page", type=int, default=1, help="1-based start page")
    parser.add_argument("--end-page", type=int, default=None, help="1-based end page")

    parser.add_argument(
        "--engine",
        choices=["auto", "marker", "surya", "text"],
        default="auto",
        help=(
            "Text extraction engine: auto=marker->surya->text fallback, "
            "marker=marker only, surya=surya only, text=built-in text layer only"
        ),
    )
    parser.add_argument(
        "--force-ocr",
        action="store_true",
        help="Force local OCR fallback (`ocrmac`) for all pages after engine step",
    )
    parser.add_argument(
        "--no-ocr", action="store_true", help="Disable local OCR fallback backend"
    )
    parser.add_argument(
        "--language",
        action="append",
        default=[],
        help="Local OCR language tag for ocrmac, repeatable (default: en-US)",
    )
    parser.add_argument(
        "--marker-force-ocr",
        action=argparse.BooleanOptionalAction,
        default=True,
        help="Pass --force_ocr to marker for higher OCR quality (default: true)",
    )
    parser.add_argument(
        "--surya-disable-math",
        action="store_true",
        help="Disable Surya math OCR mode (can reduce false positives)",
    )
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    ocr_backend = build_default_ocr_backend(
        enabled=not args.no_ocr,
        languages=args.language or ["en-US"],
    )

    config = ConversionConfig(
        input_pdf=args.input,
        latex_dir=args.latex_dir,
        markdown_dir=args.markdown_dir,
        images_dir=args.images_dir,
        dpi=args.dpi,
        start_page=args.start_page,
        end_page=args.end_page,
        engine=args.engine,
        force_ocr=args.force_ocr,
        marker_force_ocr=args.marker_force_ocr,
        surya_disable_math=args.surya_disable_math,
    )

    converter = PdfBookConverter(config=config, ocr_backend=ocr_backend)
    result = converter.convert()

    method_counts = Counter(page.extraction_method for page in result.pages)

    print(f"Input PDF: {config.input_pdf}")
    print(f"Total pages in PDF: {result.total_pdf_pages}")
    print(f"Pages converted: {result.pages_converted}")
    print(f"Engine mode: {config.engine}")
    print(f"Images output: {config.images_dir}")
    print(f"Markdown output: {config.markdown_dir}")
    print(f"LaTeX output: {config.latex_dir}")
    print("Extraction modes:")
    for method, count in sorted(method_counts.items()):
        print(f"  - {method}: {count}")
    print(f"Main Markdown: {result.main_markdown_path}")
    print(f"Main LaTeX: {result.main_latex_path}")

    if args.engine in {"auto", "marker"} and shutil.which("marker_single") is None:
        print("Note: marker not installed (`marker_single` missing).")
    if args.engine in {"auto", "surya"} and shutil.which("surya_ocr") is None:
        print("Note: surya not installed (`surya_ocr` missing).")
    if ocr_backend is None and not args.no_ocr:
        print("Note: local OCR backend unavailable; empty pages may stay empty.")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
