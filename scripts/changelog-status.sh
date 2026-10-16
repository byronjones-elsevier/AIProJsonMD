#!/usr/bin/env bash
#
# changelog-status.sh: show, for each CHANGELOG.md entry, the facts that exist only
# after a pull request is opened or merged, derived from the git history.
#
# CHANGELOG.md entries deliberately do not store the pull request number, the merge
# time, or the commit range. Those facts do not exist until after the entry is
# written, so copying them into the file would need a follow-up commit after every
# merge. This script reads them from the merge commits on the default branch instead.
# A merge commit made by GitHub has the subject "Merge pull request #<n> from
# <owner>/<branch>".
#
# For each "### <branch>: <summary>" heading in the changelog it prints one
# tab-separated row for every merge of that branch, or a single "not merged" row if no
# merge commit names it. Times are UTC (ISO 8601). Columns:
#
#   branch  status  pr  merged_utc  commits  first_commit_utc  last_commit_utc
#
# Usage: changelog-status.sh [-r REF] [-f FILE] [-t] [-h]
#   -r REF   git ref whose first-parent history is searched (default: main)
#   -f FILE  changelog file to read (default: CHANGELOG.md)
#   -t       align the columns as a table instead of printing raw tab-separated rows
#   -h       print this help and exit
#
# Exit status: 0 on success, 1 on a missing file or ref, 2 on a usage error.

set -euo pipefail

readonly TIME_FORMAT='%Y-%m-%dT%H:%M:%SZ'

usage() {
	# Print the leading comment block (everything after the shebang up to the first
	# blank line that follows it) as the help text.
	sed -n '3,/^$/p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

die() {
	local status="$1"
	shift
	printf 'changelog-status: %s\n' "$*" >&2
	exit "${status}"
}

# commit_time prints the committer time of a commit in UTC. TZ=UTC with the
# format-local date style converts to UTC; the plain format style would reuse the
# commit's own offset.
commit_time() {
	TZ=UTC git log -1 --date="format-local:${TIME_FORMAT}" --format='%cd' "$1"
}

# list_merges prints one tab-separated line per pull request merge on the first-parent
# history of the ref: <merge sha>, <pr number>, <branch>.
list_merges() {
	local ref="$1"
	local sha subject pr branch
	git log --first-parent --merges --format='%H%x09%s' "${ref}" |
		while IFS=$'\t' read -r sha subject; do
			case "${subject}" in
			"Merge pull request #"*" from "*) ;;
			*) continue ;;
			esac
			pr="${subject#Merge pull request #}"
			pr="${pr%% from *}"
			branch="${subject#* from }"
			# Drop the owner: "owner/feat/x" becomes "feat/x".
			branch="${branch#*/}"
			printf '%s\t%s\t%s\n' "${sha}" "${pr}" "${branch}"
		done
}

# describe_merge prints the row for one merge of an entry's branch.
describe_merge() {
	local branch="$1" sha="$2" pr="$3"
	local parents first_parent second_parent
	local count first last

	# A merge commit lists its parents after its own sha: <sha> <first> <second>.
	parents="$(git rev-list --parents -n 1 "${sha}")"
	read -r _ first_parent second_parent <<<"${parents}"

	# The branch's commits are those reachable from the second parent but not from the
	# first. They are listed oldest first.
	count="$(git rev-list --count "${first_parent}..${second_parent}")"
	first="$(git rev-list --reverse "${first_parent}..${second_parent}" | head -n 1)"
	last="${second_parent}"

	printf '%s\tmerged\t#%s\t%s\t%s\t%s\t%s\n' \
		"${branch}" "${pr}" "$(commit_time "${sha}")" "${count}" \
		"$(commit_time "${first}")" "$(commit_time "${last}")"
}

# changelog_rows prints the header and one or more rows for every entry heading.
changelog_rows() {
	local ref="$1" file="$2"
	local merges heading branch matches sha pr

	merges="$(list_merges "${ref}")"
	printf 'branch\tstatus\tpr\tmerged_utc\tcommits\tfirst_commit_utc\tlast_commit_utc\n'

	while IFS= read -r heading; do
		# "### feat/x: summary" names the branch "feat/x".
		branch="${heading#\#\#\# }"
		branch="${branch%%: *}"

		matches="$(printf '%s\n' "${merges}" | awk -F'\t' -v b="${branch}" '$3 == b')"
		if [[ -z "${matches}" ]]; then
			printf '%s\tnot merged\t-\t-\t-\t-\t-\n' "${branch}"
			continue
		fi
		while IFS=$'\t' read -r sha pr _; do
			describe_merge "${branch}" "${sha}" "${pr}"
		done <<<"${matches}"
	done < <(grep '^### ' "${file}" || true)
}

main() {
	local ref="main" file="CHANGELOG.md" table=0 opt

	while getopts ':r:f:th' opt; do
		case "${opt}" in
		r) ref="${OPTARG}" ;;
		f) file="${OPTARG}" ;;
		t) table=1 ;;
		h)
			usage
			exit 0
			;;
		:) die 2 "option -${OPTARG} needs an argument" ;;
		*) die 2 "unknown option -${OPTARG}" ;;
		esac
	done

	git rev-parse --git-dir >/dev/null 2>&1 || die 1 "not inside a git repository"
	[[ -f "${file}" ]] || die 1 "changelog file not found: ${file}"
	git rev-parse --verify --quiet "${ref}^{commit}" >/dev/null || die 1 "unknown git ref: ${ref}"

	if ((table)); then
		changelog_rows "${ref}" "${file}" | column -t -s "$(printf '\t')"
	else
		changelog_rows "${ref}" "${file}"
	fi
}

main "$@"
