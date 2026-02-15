from __future__ import annotations

from pathlib import Path

import fitz

from taocp_converter.converter import (
    ConversionConfig,
    PdfBookConverter,
    compact_page_range,
    parse_marker_paginated_markdown,
    parse_surya_results,
    text_to_latex,
)


class FakeOCRBackend:
    def __init__(self, text: str) -> None:
        self._text = text

    def extract_text(self, image_path: Path) -> str:
        return self._text


def _make_text_pdf(path: Path) -> None:
    doc = fitz.open()

    page1 = doc.new_page()
    page1.insert_text(
        (72, 72),
        "Page 1 line A\\nPage 1 line B with symbols: & % $ # _ { } ~ ^ \\\",
    )

    page2 = doc.new_page()
    page2.insert_text((72, 72), "Page 2 content")

    doc.save(path)
    doc.close()


def _make_empty_pdf(path: Path) -> None:
    doc = fitz.open()
    doc.new_page()
    doc.save(path)
    doc.close()


def test_compact_page_range() -> None:
    assert compact_page_range([0, 1, 2, 5, 6, 9]) == "0-2,5-6,9"
    assert compact_page_range([3]) == "3"
    assert compact_page_range([]) == ""


def test_parse_marker_paginated_markdown() -> None:
    markdown = (
        "\\n\\n{0}------------------------------------------------\\n\\n"
        "First page text.\\n\\n"
        "{1}------------------------------------------------\\n\\n"
        "Second page text.\\n"
    )
    parsed = parse_marker_paginated_markdown(markdown)
    assert parsed[1] == "First page text."
    assert parsed[2] == "Second page text."


def test_parse_surya_results() -> None:
    results = {
        "book": [
            {"text_lines": [{"text": "A"}, {"text": "B"}]},
            {"text_lines": [{"text": "C"}]},
        ]
    }
    parsed = parse_surya_results(results, requested_pages=[4, 5])
    assert parsed[4] == "A\\nB"
    assert parsed[5] == "C"


def test_text_to_latex_escapes_special_characters() -> None:
    rendered = text_to_latex(r"& % $ # _ { } ~ ^ ")
    assert r"\&" in rendered
    assert r"\%" in rendered
    assert r"\$" in rendered
    assert r"\#" in rendered
    assert r"\_" in rendered
    assert r"\{" in rendered
    assert r"\}" in rendered
    assert r"\textasciitilde{}" in rendered
    assert r"\textasciicircum{}" in rendered
    assert r"\textbackslash{}" in rendered


def test_converter_writes_markdown_latex_and_images(tmp_path: Path) -> None:
    input_pdf = tmp_path / "book.pdf"
    _make_text_pdf(input_pdf)

    config = ConversionConfig(
        input_pdf=input_pdf,
        latex_dir=tmp_path / "latex",
        markdown_dir=tmp_path / "markdown",
        images_dir=tmp_path / "images",
        dpi=150,
        engine="text",
    )

    result = PdfBookConverter(config=config, ocr_backend=None).convert()

    assert result.pages_converted == 2

    assert (tmp_path / "images" / "page_0001.png").exists()
    assert (tmp_path / "images" / "page_0002.png").exists()

    page_md = (tmp_path / "markdown" / "page_0001.md").read_text(encoding="utf-8")
    assert "# Page 1" in page_md
    assert "Page 1 line A" in page_md
    assert "../images/page_0001.png" in page_md

    page_tex = (tmp_path / "latex" / "page_0001.tex").read_text(encoding="utf-8")
    assert r"\section*{Page 1}" in page_tex
    assert r"\includegraphics" in page_tex
    assert r"\&" in page_tex

    main_md = (tmp_path / "markdown" / "main.md").read_text(encoding="utf-8")
    assert "[Page 1](page_0001.md)" in main_md
    assert "[Page 2](page_0002.md)" in main_md

    main_tex = (tmp_path / "latex" / "main.tex").read_text(encoding="utf-8")
    assert r"\input{page_0001.tex}" in main_tex
    assert r"\input{page_0002.tex}" in main_tex


def test_converter_uses_ocr_when_text_layer_empty(tmp_path: Path) -> None:
    input_pdf = tmp_path / "empty.pdf"
    _make_empty_pdf(input_pdf)

    config = ConversionConfig(
        input_pdf=input_pdf,
        latex_dir=tmp_path / "latex",
        markdown_dir=tmp_path / "markdown",
        images_dir=tmp_path / "images",
        dpi=150,
        engine="text",
    )

    ocr_backend = FakeOCRBackend("OCR recovered text")
    result = PdfBookConverter(config=config, ocr_backend=ocr_backend).convert()

    assert result.pages_converted == 1
    assert result.pages[0].extraction_method == "ocrmac"

    page_md = (tmp_path / "markdown" / "page_0001.md").read_text(encoding="utf-8")
    assert "OCR recovered text" in page_md
