from __future__ import annotations

from dataclasses import dataclass
import json
import os
from os.path import relpath
from pathlib import Path
import platform
import re
import shlex
import shutil
import subprocess
import tempfile
import time
from typing import Protocol, Sequence

import fitz


MARKER_PAGE_BREAK_RE = re.compile(r"\{(?P<page>\d+)\}\s*-{20,}\s*")
MATH_TAG_RE = re.compile(r"<math>(.*?)</math>", re.S | re.I)
SUB_TAG_RE = re.compile(r"<sub>(.*?)</sub>", re.S | re.I)
SUP_TAG_RE = re.compile(r"<sup>(.*?)</sup>", re.S | re.I)


class OCRBackend(Protocol):
    """Extract text from a rendered page image."""

    def extract_text(self, image_path: Path) -> str:
        ...


class OcrMacBackend:
    """OCR backend using macOS Vision framework via `ocrmac`."""

    def __init__(self, languages: Sequence[str] | None = None) -> None:
        from ocrmac import ocrmac as ocrmac_module

        self._ocrmac = ocrmac_module
        self._languages = tuple(languages or ["en-US"])

    def extract_text(self, image_path: Path) -> str:
        annotations = self._ocrmac.OCR(
            str(image_path), language_preference=list(self._languages)
        ).recognize(px=False)
        lines: list[str] = []
        for annotation in annotations:
            if not annotation:
                continue
            text = str(annotation[0]).strip()
            if text:
                lines.append(text)
        return "\n".join(lines)


@dataclass(slots=True)
class ConversionConfig:
    input_pdf: Path
    latex_dir: Path
    markdown_dir: Path
    images_dir: Path
    dpi: int = 220
    start_page: int = 1
    end_page: int | None = None
    engine: str = "surya"  # surya (default) | marker | auto | text
    force_ocr: bool = False
    marker_force_ocr: bool = True
    surya_disable_math: bool = False
    verbose: bool = True


@dataclass(slots=True)
class PageResult:
    page_number: int
    image_path: Path
    markdown_path: Path
    latex_path: Path
    extraction_method: str
    text_characters: int


@dataclass(slots=True)
class ConversionResult:
    total_pdf_pages: int
    pages_converted: int
    pages: list[PageResult]
    main_markdown_path: Path
    main_latex_path: Path


def build_default_ocr_backend(
    enabled: bool = True, languages: Sequence[str] | None = None
) -> OCRBackend | None:
    """Build a default OCR backend for the current platform."""
    if not enabled:
        return None
    if platform.system() == "Darwin":
        try:
            return OcrMacBackend(languages=languages)
        except Exception:
            return None
    return None


def normalize_text(text: str) -> str:
    text = text.replace("\r\n", "\n").replace("\r", "\n")
    # Drop non-printable control characters that can leak from malformed
    # PDF text layers and break markdown/LaTeX consumers.
    text = re.sub(r"[\x00-\x08\x0b-\x1f\x7f]", "", text)
    lines = [line.rstrip() for line in text.split("\n")]
    compact = "\n".join(lines)
    compact = re.sub(r"\n{3,}", "\n\n", compact)
    return compact.strip()


def escape_latex(text: str) -> str:
    escaped: list[str] = []
    char_map = {
        "&": r"\&",
        "%": r"\%",
        "$": r"\$",
        "#": r"\#",
        "_": r"\_",
        "{": r"\{",
        "}": r"\}",
        "~": r"\textasciitilde{}",
        "^": r"\textasciicircum{}",
        "\\": r"\textbackslash{}",
    }
    for char in text:
        escaped.append(char_map.get(char, char))
    return "".join(escaped)


def text_to_latex(text: str) -> str:
    text = normalize_text(text)
    if not text:
        return "% No extracted text for this page."

    paragraphs = [p.strip() for p in re.split(r"\n\s*\n", text) if p.strip()]
    rendered = [escape_latex(" ".join(p.splitlines())) for p in paragraphs]
    return "\n\n".join(rendered)


def compact_page_range(pages_zero_based: Sequence[int]) -> str:
    """Convert page indices like [0,1,2,5,6] -> "0-2,5-6"."""
    if not pages_zero_based:
        return ""

    sorted_pages = sorted(set(pages_zero_based))
    ranges: list[str] = []
    start = sorted_pages[0]
    prev = sorted_pages[0]

    for page in sorted_pages[1:]:
        if page == prev + 1:
            prev = page
            continue
        if start == prev:
            ranges.append(str(start))
        else:
            ranges.append(f"{start}-{prev}")
        start = page
        prev = page

    if start == prev:
        ranges.append(str(start))
    else:
        ranges.append(f"{start}-{prev}")

    return ",".join(ranges)


def parse_marker_paginated_markdown(markdown: str) -> dict[int, str]:
    """Parse marker's paginated markdown into a page_number -> text mapping."""
    matches = list(MARKER_PAGE_BREAK_RE.finditer(markdown))
    if not matches:
        return {}

    by_page: dict[int, str] = {}
    for index, match in enumerate(matches):
        start = match.end()
        end = matches[index + 1].start() if index + 1 < len(matches) else len(markdown)
        # marker emits 0-based page IDs (global page index), so convert to 1-based
        # page numbers for converter outputs.
        page_number = int(match.group("page")) + 1
        text = normalize_text(markdown[start:end])
        by_page[page_number] = text

    return by_page


def parse_surya_results(results: dict, requested_pages: Sequence[int]) -> dict[int, str]:
    if not results:
        return {}

    first_value = next(iter(results.values()), [])
    if not isinstance(first_value, list):
        return {}

    by_page: dict[int, str] = {}
    for index, page in enumerate(first_value):
        if index >= len(requested_pages):
            break
        lines = page.get("text_lines", [])
        page_text_lines = [
            str(line.get("text", "")).strip()
            for line in lines
            if str(line.get("text", "")).strip()
        ]
        by_page[requested_pages[index]] = normalize_text("\n".join(page_text_lines))

    return by_page


def should_retry_marker_on_cpu(error_output: str) -> bool:
    text = error_output.lower()
    return (
        "torch.acceleratorerror" in text
        or "mps" in text
        or "index 4096 is out of bounds" in text
    )


def normalize_markdown_math(text: str) -> str:
    """Convert common HTML math markup into markdown-friendly LaTeX delimiters."""

    def _inline_math(match: re.Match[str]) -> str:
        payload = normalize_text(match.group(1))
        return f"${payload}$" if payload else ""

    def _subscript(match: re.Match[str]) -> str:
        payload = normalize_text(match.group(1))
        return f"_{{{payload}}}" if payload else ""

    def _superscript(match: re.Match[str]) -> str:
        payload = normalize_text(match.group(1))
        return f"^{{{payload}}}" if payload else ""

    text = MATH_TAG_RE.sub(_inline_math, text)
    text = SUB_TAG_RE.sub(_subscript, text)
    text = SUP_TAG_RE.sub(_superscript, text)
    return text


class PdfBookConverter:
    def __init__(self, config: ConversionConfig, ocr_backend: OCRBackend | None = None) -> None:
        self.config = config
        self.ocr_backend = ocr_backend

    def _log(self, message: str) -> None:
        if self.config.verbose:
            print(f"[taocp-convert] {message}", flush=True)

    @staticmethod
    def _format_command(command: Sequence[str]) -> str:
        return " ".join(shlex.quote(part) for part in command)

    def _run_external_command(
        self,
        command: Sequence[str],
        label: str,
        env_overrides: dict[str, str] | None = None,
    ) -> str:
        command_str = self._format_command(command)
        if env_overrides:
            env_display = ", ".join(f"{key}={value}" for key, value in env_overrides.items())
            command_str = f"{env_display} {command_str}"
        self._log(f"RUN[{label}] {command_str}")
        start = time.time()
        run_env = os.environ.copy()
        if env_overrides:
            run_env.update(env_overrides)

        if self.config.verbose:
            process = subprocess.Popen(
                command,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                text=True,
                bufsize=1,
                env=run_env,
            )
            output_lines: list[str] = []
            assert process.stdout is not None
            for line in process.stdout:
                output_lines.append(line)
                text = line.rstrip()
                if text:
                    self._log(f"[{label}] {text}")
            process.wait()
            output = "".join(output_lines)
            duration = time.time() - start
            self._log(f"DONE[{label}] exit={process.returncode} duration={duration:.2f}s")
            if process.returncode != 0:
                raise RuntimeError(
                    f"{label} failed with exit code {process.returncode}. "
                    f"Last output:\n{output[-2000:]}"
                )
            return output

        completed = subprocess.run(
            command, capture_output=True, text=True, check=False, env=run_env
        )
        duration = time.time() - start
        self._log(f"DONE[{label}] exit={completed.returncode} duration={duration:.2f}s")
        output = completed.stdout + completed.stderr
        if completed.returncode != 0:
            raise RuntimeError(
                f"{label} failed with exit code {completed.returncode}. "
                f"Last output:\n{output[-2000:]}"
            )
        return output

    def convert(self) -> ConversionResult:
        self._validate_config()
        self._ensure_dirs()
        start_time = time.time()
        self._log(f"Input PDF: {self.config.input_pdf}")
        self._log(
            "Output dirs: "
            f"images={self.config.images_dir}, "
            f"markdown={self.config.markdown_dir}, "
            f"latex={self.config.latex_dir}"
        )
        self._log(
            f"Options: engine={self.config.engine}, dpi={self.config.dpi}, "
            f"force_ocr={self.config.force_ocr}, marker_force_ocr={self.config.marker_force_ocr}"
        )

        with fitz.open(self.config.input_pdf) as document:
            total_pages = len(document)
            start_page, end_page = self._resolve_page_bounds(total_pages)
            requested_pages = list(range(start_page, end_page + 1))
            self._log(
                f"Processing pages {start_page}-{end_page} "
                f"(count={len(requested_pages)}, total_pdf_pages={total_pages})"
            )

            hq_texts, hq_methods = self._extract_high_quality_text(requested_pages)

            pages: list[PageResult] = []
            for index, page_number in enumerate(requested_pages, start=1):
                self._log(f"[{index}/{len(requested_pages)}] Converting page {page_number}")
                page = document.load_page(page_number - 1)
                image_path = self._write_page_image(page, page_number)

                text = normalize_text(hq_texts.get(page_number, ""))
                extraction_method = hq_methods.get(page_number, "")

                if not text:
                    text = normalize_text(page.get_text("text"))
                    extraction_method = "text-layer" if text else "empty"

                should_try_local_ocr = self.config.force_ocr or (not text)
                if should_try_local_ocr and self.ocr_backend is not None:
                    self._log(f"Running local OCR fallback on page {page_number}")
                    ocr_text = normalize_text(self.ocr_backend.extract_text(image_path))
                    if ocr_text:
                        text = ocr_text
                        extraction_method = "ocrmac"

                markdown_path = self._write_markdown_page(page_number, image_path, text, extraction_method)
                latex_path = self._write_latex_page(page_number, image_path, text, extraction_method)
                self._log(
                    f"Finished page {page_number}: method={extraction_method}, chars={len(text)}, "
                    f"image={image_path.name}, markdown={markdown_path.name}, latex={latex_path.name}"
                )

                pages.append(
                    PageResult(
                        page_number=page_number,
                        image_path=image_path,
                        markdown_path=markdown_path,
                        latex_path=latex_path,
                        extraction_method=extraction_method,
                        text_characters=len(text),
                    )
                )

        main_markdown = self._write_markdown_main(pages)
        main_latex = self._write_latex_main(pages)
        self._log(f"Wrote {main_markdown}")
        self._log(f"Wrote {main_latex}")
        self._log(f"Completed in {time.time() - start_time:.2f}s")

        return ConversionResult(
            total_pdf_pages=total_pages,
            pages_converted=len(pages),
            pages=pages,
            main_markdown_path=main_markdown,
            main_latex_path=main_latex,
        )

    def _resolve_page_bounds(self, total_pages: int) -> tuple[int, int]:
        start_page = self.config.start_page
        end_page = self.config.end_page or total_pages

        if start_page < 1 or start_page > total_pages:
            raise ValueError(f"start_page must be between 1 and {total_pages}, got {start_page}")
        if end_page < start_page or end_page > total_pages:
            raise ValueError(
                f"end_page must be between {start_page} and {total_pages}, got {end_page}"
            )
        return start_page, end_page

    def _extract_high_quality_text(
        self, page_numbers: Sequence[int]
    ) -> tuple[dict[int, str], dict[int, str]]:
        if self.config.engine not in {"auto", "marker", "surya", "text"}:
            raise ValueError(
                f"engine must be one of auto|marker|surya|text, got {self.config.engine}"
            )

        texts: dict[int, str] = {}
        methods: dict[int, str] = {}
        remaining = set(page_numbers)

        if self.config.engine in {"auto", "surya"}:
            self._log(f"High-quality step: trying surya on {len(remaining)} pages")
            try:
                surya_texts = self._extract_with_surya(sorted(remaining))
                for page, text in surya_texts.items():
                    if text:
                        texts[page] = text
                        methods[page] = "surya"
                        remaining.discard(page)
                self._log(f"Surya filled {len(surya_texts)} pages")
            except RuntimeError as error:
                if self.config.engine == "surya":
                    raise
                self._log(f"Surya unavailable/failed: {error}")
                self._log("Continuing to next engine")

        if self.config.engine in {"auto", "marker"} and remaining:
            self._log(f"High-quality step: trying marker on {len(remaining)} pages")
            try:
                marker_texts = self._extract_with_marker(sorted(remaining))
                for page, text in marker_texts.items():
                    if text:
                        texts[page] = text
                        methods[page] = "marker"
                        remaining.discard(page)
                self._log(f"Marker filled {len(marker_texts)} pages")
            except RuntimeError as error:
                if self.config.engine == "marker":
                    raise
                self._log(f"Marker unavailable/failed: {error}")
                self._log("Falling back to local extraction")

        return texts, methods

    def _extract_with_marker(self, page_numbers: Sequence[int]) -> dict[int, str]:
        marker_cmd = shutil.which("marker_single")
        if marker_cmd is None:
            raise RuntimeError(
                "marker_single is not available. Install with: uv add marker-pdf"
            )

        page_range = compact_page_range([page - 1 for page in page_numbers])
        self._log(
            f"Marker config: page_range={page_range or 'all'}, "
            f"force_ocr={self.config.marker_force_ocr}"
        )
        with tempfile.TemporaryDirectory(prefix="marker_run_") as temp_dir:
            command = [
                marker_cmd,
                str(self.config.input_pdf),
                "--output_dir",
                temp_dir,
                "--output_format",
                "markdown",
                "--paginate_output",
                "--disable_image_extraction",
            ]
            if page_range:
                command.extend(["--page_range", page_range])
            if self.config.marker_force_ocr:
                command.append("--force_ocr")
            if self.config.verbose:
                command.append("--debug")

            try:
                self._run_external_command(command, "marker_single")
            except RuntimeError as error:
                current_device = os.environ.get("TORCH_DEVICE", "").lower()
                if current_device == "cpu" or not should_retry_marker_on_cpu(str(error)):
                    raise
                self._log(
                    "Marker failed on accelerator backend; retrying marker_single with TORCH_DEVICE=cpu"
                )
                self._run_external_command(
                    command,
                    "marker_single",
                    env_overrides={"TORCH_DEVICE": "cpu"},
                )

            stem = self.config.input_pdf.stem
            markdown_path = Path(temp_dir) / stem / f"{stem}.md"
            if not markdown_path.exists():
                raise RuntimeError(f"marker output markdown not found at {markdown_path}")
            self._log(f"Marker output markdown: {markdown_path}")

            markdown = markdown_path.read_text(encoding="utf-8")

        parsed = parse_marker_paginated_markdown(markdown)
        if not parsed and len(page_numbers) == 1:
            single_page = page_numbers[0]
            text = normalize_text(markdown)
            return {single_page: text}

        return {
            page: text
            for page, text in parsed.items()
            if page in page_numbers and normalize_text(text)
        }

    def _extract_with_surya(self, page_numbers: Sequence[int]) -> dict[int, str]:
        surya_cmd = shutil.which("surya_ocr")
        if surya_cmd is None:
            raise RuntimeError("surya_ocr is not available. Install with: uv add surya-ocr")

        page_range = compact_page_range([page - 1 for page in page_numbers])
        self._log(
            f"Surya config: page_range={page_range or 'all'}, "
            f"disable_math={self.config.surya_disable_math}"
        )
        with tempfile.TemporaryDirectory(prefix="surya_run_") as temp_dir:
            command = [
                surya_cmd,
                str(self.config.input_pdf),
                "--output_dir",
                temp_dir,
            ]
            if page_range:
                command.extend(["--page_range", page_range])
            if self.config.surya_disable_math:
                command.append("--disable_math")
            if self.config.verbose:
                command.append("--debug")

            self._run_external_command(command, "surya_ocr")

            results_path = Path(temp_dir) / self.config.input_pdf.stem / "results.json"
            if not results_path.exists():
                raise RuntimeError(f"surya results not found at {results_path}")
            self._log(f"Surya results JSON: {results_path}")

            results = json.loads(results_path.read_text(encoding="utf-8"))

        parsed = parse_surya_results(results, page_numbers)
        return {page: text for page, text in parsed.items() if normalize_text(text)}

    def _validate_config(self) -> None:
        if not self.config.input_pdf.exists():
            raise FileNotFoundError(f"Input PDF does not exist: {self.config.input_pdf}")
        if self.config.dpi <= 0:
            raise ValueError(f"dpi must be > 0, got {self.config.dpi}")

    def _ensure_dirs(self) -> None:
        self.config.latex_dir.mkdir(parents=True, exist_ok=True)
        self.config.markdown_dir.mkdir(parents=True, exist_ok=True)
        self.config.images_dir.mkdir(parents=True, exist_ok=True)

    def _write_page_image(self, page: fitz.Page, page_number: int) -> Path:
        zoom = self.config.dpi / 72.0
        pixmap = page.get_pixmap(matrix=fitz.Matrix(zoom, zoom), alpha=False)
        image_path = self.config.images_dir / f"page_{page_number:04d}.png"
        pixmap.save(image_path)
        return image_path

    def _write_markdown_page(
        self, page_number: int, image_path: Path, text: str, extraction_method: str
    ) -> Path:
        markdown_path = self.config.markdown_dir / f"page_{page_number:04d}.md"
        image_ref = Path(relpath(image_path, start=markdown_path.parent)).as_posix()
        body = normalize_markdown_math(text) if text else "_No text extracted for this page._"
        content = (
            f"# Page {page_number}\n\n"
            f"![Page {page_number}]({image_ref})\n\n"
            f"<!-- extraction: {extraction_method} -->\n\n"
            f"{body}\n"
        )
        markdown_path.write_text(content, encoding="utf-8")
        return markdown_path

    def _write_latex_page(
        self, page_number: int, image_path: Path, text: str, extraction_method: str
    ) -> Path:
        latex_path = self.config.latex_dir / f"page_{page_number:04d}.tex"
        image_ref = Path(relpath(image_path, start=latex_path.parent)).as_posix()
        latex_text = text_to_latex(text)
        content = (
            f"% Auto-generated from page {page_number}\n"
            f"% extraction: {extraction_method}\n"
            f"\\section*{{Page {page_number}}}\n"
            f"\\addcontentsline{{toc}}{{section}}{{Page {page_number}}}\n"
            f"\\begin{{figure}}[htbp]\n"
            f"\\centering\n"
            f"\\includegraphics[width=0.95\\textwidth]{{{image_ref}}}\n"
            f"\\end{{figure}}\n\n"
            f"{latex_text}\n\n"
            f"\\clearpage\n"
        )
        latex_path.write_text(content, encoding="utf-8")
        return latex_path

    def _write_markdown_main(self, pages: Sequence[PageResult]) -> Path:
        main_path = self.config.markdown_dir / "main.md"
        lines = ["# Converted Book", "", "## Pages", ""]
        for page in pages:
            filename = page.markdown_path.name
            lines.append(f"- [Page {page.page_number}]({filename})")
        lines.append("")
        main_path.write_text("\n".join(lines), encoding="utf-8")
        return main_path

    def _write_latex_main(self, pages: Sequence[PageResult]) -> Path:
        main_path = self.config.latex_dir / "main.tex"
        lines = [
            "\\documentclass[11pt]{book}",
            "\\usepackage[T1]{fontenc}",
            "\\usepackage[utf8]{inputenc}",
            "\\usepackage[a4paper,margin=1in]{geometry}",
            "\\usepackage{graphicx}",
            "\\usepackage{hyperref}",
            "\\begin{document}",
            "\\tableofcontents",
            "",
        ]
        for page in pages:
            lines.append(f"\\input{{{page.latex_path.name}}}")
        lines.extend(["", "\\end{document}", ""])
        main_path.write_text("\n".join(lines), encoding="utf-8")
        return main_path
