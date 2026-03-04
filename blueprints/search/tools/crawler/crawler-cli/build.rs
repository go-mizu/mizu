use std::process::Command;

fn main() {
    // Trigger rebuild when git HEAD or tags change.
    if let Some(git_dir) = git_cmd(&["rev-parse", "--git-dir"]) {
        println!("cargo:rerun-if-changed={git_dir}/HEAD");
        println!("cargo:rerun-if-changed={git_dir}/packed-refs");
    }

    // Version: `git describe --tags --dirty --match "v*"`, same pattern as Go search binary.
    // Falls back to CARGO_PKG_VERSION if no matching tag exists.
    let version = git_cmd(&["describe", "--tags", "--dirty", "--match", "v*"])
        .unwrap_or_else(|| format!("v{}", env!("CARGO_PKG_VERSION")));

    // Short commit hash.
    let commit = git_cmd(&["rev-parse", "--short", "HEAD"])
        .unwrap_or_else(|| "unknown".to_string());

    // UTC build timestamp (ISO 8601).
    let build_time = date_utc().unwrap_or_else(|| "unknown".to_string());

    println!("cargo:rustc-env=CRAWLER_GIT_VERSION={version}");
    println!("cargo:rustc-env=CRAWLER_GIT_COMMIT={commit}");
    println!("cargo:rustc-env=CRAWLER_BUILD_TIME={build_time}");
}

/// Run a git command and return trimmed stdout, or None on failure.
fn git_cmd(args: &[&str]) -> Option<String> {
    Command::new("git")
        .args(args)
        .output()
        .ok()
        .filter(|o| o.status.success())
        .map(|o| String::from_utf8_lossy(&o.stdout).trim().to_string())
        .filter(|s| !s.is_empty())
}

/// Return UTC timestamp in ISO 8601 format (e.g. "2026-03-02T04:00:00Z").
fn date_utc() -> Option<String> {
    Command::new("date")
        .args(["-u", "+%Y-%m-%dT%H:%M:%SZ"])
        .output()
        .ok()
        .filter(|o| o.status.success())
        .map(|o| String::from_utf8_lossy(&o.stdout).trim().to_string())
}
