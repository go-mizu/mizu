#!/bin/sh
#
# Tag a release across every module in the repository.
#
#   scripts/tag-release.sh v0.7.0
#   scripts/tag-release.sh v0.7.0 --dry-run
#
# The repository holds more than one Go module. The toolkit is the one at the
# root, and each tool under tools/ is its own module so that its dependencies
# stay out of everybody else's go.sum. Go versions a nested module by a tag
# that carries its directory as a prefix, so tools/milestonebot v0.7.0 is the
# tag tools/milestonebot/v0.7.0.
#
# They all move together on one version. That is a choice, not a requirement:
# separate versions would be more precise and would mean working out which of
# five numbers to bump every time, and nobody wants that conversation. One
# number for the repository is easier to talk about and easier to check.
#
# The script refuses to tag anything it is not sure about. It wants a clean
# tree on main, a tag that does not exist yet, a version that parses, and a
# test suite that passes. Pushing the tags is the last thing it does.

set -eu

usage() {
	echo "usage: scripts/tag-release.sh vX.Y.Z[-pre] [--dry-run]" >&2
	exit 2
}

version=""
dry_run=no
for arg in "$@"; do
	case "$arg" in
	--dry-run) dry_run=yes ;;
	-h | --help) usage ;;
	-*) echo "unknown flag $arg" >&2 && usage ;;
	*)
		[ -n "$version" ] && usage
		version="$arg"
		;;
	esac
done
[ -n "$version" ] || usage

# v1.2.3, or v1.2.3-rc.1 for a prerelease. Anything else is a typo.
case "$version" in
v[0-9]*.[0-9]*.[0-9]*) ;;
*)
	echo "version must look like v1.2.3, got $version" >&2
	exit 1
	;;
esac

cd "$(dirname "$0")/.."

run() {
	if [ "$dry_run" = yes ]; then
		echo "would run: $*"
	else
		"$@"
	fi
}

branch=$(git rev-parse --abbrev-ref HEAD)
if [ "$branch" != main ]; then
	echo "on branch $branch, releases are tagged from main" >&2
	exit 1
fi

if [ -n "$(git status --porcelain)" ]; then
	echo "the working tree has changes, and a tag should point at something that exists elsewhere" >&2
	git status --short >&2
	exit 1
fi

git fetch --tags --quiet
if [ "$(git rev-parse HEAD)" != "$(git rev-parse '@{upstream}')" ]; then
	echo "HEAD and the upstream branch disagree, push or pull first" >&2
	exit 1
fi

# Every directory holding a go.mod is a module in the train, except the ones
# under testdata, which exist to be read by a test and are never published.
# The root module is written as "." so that it survives word splitting, and
# it is the one module tagged by the bare version.
modules=$(git ls-files '*go.mod' |
	grep -v '/testdata/' |
	sed -e 's|/\{0,1\}go.mod$||' -e 's|^$|.|')

tags=""
for module in $modules; do
	if [ "$module" = . ]; then
		tags="$tags $version"
	else
		tags="$tags $module/$version"
	fi
done

for tag in $tags; do
	if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
		echo "tag $tag already exists" >&2
		exit 1
	fi
done

echo "Checking every module before tagging."
for module in $modules; do
	echo "  $module"
	unformatted=$(cd "$module" && gofmt -l .)
	if [ -n "$unformatted" ]; then
		echo "gofmt would change these files in $module:" >&2
		echo "$unformatted" >&2
		exit 1
	fi
	(cd "$module" && go vet ./... && go test ./...)
done

echo
echo "Tagging:"
for tag in $tags; do
	echo "  $tag"
	run git tag -a "$tag" -m "$tag"
done

echo
echo "Pushing."
# One push so the tags land together. A release built from half a train is
# worse than one that never started.
# shellcheck disable=SC2086
run git push origin $tags

echo
echo "Done. The release workflow takes it from here."
