#!/usr/bin/env bash
# Create a module tag only after go.mod / nested-module checks pass.
# Releases are created only with this script. Do not run git tag by hand.
set -euo pipefail

usage() {
	echo "Usage: scripts/release.sh [--dry-run] <tag>" >&2
	echo "  tag: vX.Y.Z (root module) or path/vX.Y.Z (nested module)" >&2
	exit 2
}

dry=0
tag=""
for arg in "$@"; do
	case "$arg" in
	--dry-run) dry=1 ;;
	-h | --help) usage ;;
	*)
		if [[ -n "$tag" ]]; then
			usage
		fi
		tag="$arg"
		;;
	esac
done
[[ -n "$tag" ]] || usage

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

step() { echo "==> $*"; }

is_semver() {
	[[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

[[ -f go.mod ]] || {
	echo "missing go.mod" >&2
	exit 1
}

module="$(awk '/^module / { print $2; exit }' go.mod)"
step "go.mod module: ${module}"

nested_ok=1
step "nested modules present:"
nested_count=0
while IFS= read -r gom; do
	[[ -z "$gom" ]] && continue
	rel="${gom#./}"
	dir="$(dirname "$rel")"
	nmod="$(awk '/^module / { print $2; exit }' "$gom")"
	echo "  ${dir}  (${nmod})"
	if [[ ! -d "$dir" ]]; then
		echo "missing nested module directory: ${dir}" >&2
		nested_ok=0
	fi
	nested_count=$((nested_count + 1))
done < <(find . -mindepth 2 -name go.mod | sed 's|^\./||' | sort)
if [[ "$nested_count" -eq 0 ]]; then
	echo "  (none)"
fi
[[ "$nested_ok" -eq 1 ]] || exit 1

kind=""
if [[ "$tag" == */v[0-9]* ]]; then
	kind="nested"
	ver="${tag##*/}"
	rel="${tag%/"${ver}"}"
	step "validate nested tag format: ${rel}/${ver}"
	if ! is_semver "$ver"; then
		echo "invalid nested version in tag ${tag} (want path/vX.Y.Z)" >&2
		exit 1
	fi
	if [[ ! -f "${rel}/go.mod" ]]; then
		echo "nested module missing: ${rel}/go.mod" >&2
		exit 1
	fi
	nmod="$(awk '/^module / { print $2; exit }' "${rel}/go.mod")"
	want="${module}/${rel}"
	if [[ "$nmod" != "$want" ]]; then
		echo "go.mod module ${nmod} does not match expected ${want}" >&2
		exit 1
	fi
else
	kind="root"
	ver="$tag"
	step "validate root tag format: ${ver}"
	if ! is_semver "$ver"; then
		echo "invalid root tag ${tag} (want vX.Y.Z)" >&2
		exit 1
	fi
	if [[ "$module" == */v2 ]]; then
		if [[ "$ver" != v2.* ]]; then
			echo "module ${module} requires tags v2.Y.Z, got ${ver}" >&2
			exit 1
		fi
	elif [[ "$ver" == v2.* ]]; then
		echo "v1 module ${module} must not be tagged ${ver}" >&2
		exit 1
	fi
	step "check go.mod requires of nested paths"
	req_found=0
	while read -r req; do
		[[ -z "$req" ]] && continue
		[[ "$req" == "${module}/"* ]] || continue
		rel="${req#"${module}/"}"
		req_found=1
		if [[ ! -f "${rel}/go.mod" ]]; then
			echo "go.mod requires ${req} but ${rel}/go.mod is missing" >&2
			exit 1
		fi
		echo "  require ${req} -> ${rel}/go.mod ok"
	done < <(awk '
		/^require \(/ { inreq=1; next }
		inreq && /^\)/ { inreq=0; next }
		inreq { print $1 }
		/^require / && $2 != "(" { print $2 }
	' go.mod)
	if [[ "$req_found" -eq 0 ]]; then
		echo "  (no nested-path requires)"
	fi
fi

if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null; then
	echo "tag already exists: ${tag}" >&2
	exit 1
fi
step "tag ${tag} is free (${kind})"

if [[ "$dry" -eq 1 ]]; then
	step "dry-run: would run: git tag -a ${tag} -m ${tag}"
	step "dry-run: would run: git push origin ${tag}"
	echo "ok"
	exit 0
fi

step "git tag -a ${tag}"
git tag -a "$tag" -m "$tag"
step "git push origin ${tag}"
git push origin "$tag"
