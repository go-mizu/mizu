fn main() {
    // Header file is manually maintained at ../include/fts_rust.h
    // cbindgen requires nightly for expansion, so we skip auto-generation
    println!("cargo:rerun-if-changed=src/");
}
