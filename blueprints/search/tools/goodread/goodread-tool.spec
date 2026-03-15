# -*- mode: python ; coding: utf-8 -*-
# PyInstaller spec for goodread-tool self-contained binary.
# Build: uv run pyinstaller --clean --noconfirm goodread-tool.spec
from PyInstaller.utils.hooks import collect_all, collect_submodules

datas = []
binaries = []
hiddenimports = []

for pkg in ('duckdb', 'patchright', 'faker', 'httpx', 'typer', 'rich'):
    tmp = collect_all(pkg)
    datas    += tmp[0]
    binaries += tmp[1]
    hiddenimports += tmp[2]

hiddenimports += collect_submodules('goodread_tool')

a = Analysis(
    ['goodread_entry.py'],
    pathex=['src'],
    binaries=binaries,
    datas=datas,
    hiddenimports=hiddenimports,
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=['tkinter', 'matplotlib', 'numpy', 'pandas', 'scipy', 'clickhouse_connect'],
    noarchive=False,
)
pyz = PYZ(a.pure)
exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.datas,
    [],
    name='goodread-tool',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=False,
    console=True,
    disable_windowed_traceback=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
)
