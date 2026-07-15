# Maintaining the Aiven Telegraf fork

This repository is Aiven's fork of
[influxdata/telegraf](https://github.com/influxdata/telegraf). We carry a small
set of patches on top of upstream release tags. This guide covers how those
patches are organised, how changes get reviewed and merged, and how the series
moves to a new upstream version.

`AIVEN_CHANGES.md` explains why each patch exists. The ordered list of patches
is the git history itself:

```bash
git log --reverse --no-merges vX.Y.Z-upstream..vX.Y.Z-aiven
```

## Branches

Each upstream version gets two branches:

- `vX.Y.Z-upstream` is a pristine copy of the upstream tag. Nothing of ours
  lands here. It exists to be a clean, stable merge-base.
- `vX.Y.Z-aiven` is the release branch: upstream plus our patches. Builds come
  from this branch, and every PR targets it.

Nobody commits to `-aiven` directly. Every change lands through a PR whose base
is `vX.Y.Z-aiven` and whose work branch starts from `vX.Y.Z-upstream` (for a
port) or from the current `-aiven` tip (for later changes). Since `-aiven`
starts out identical to `-upstream`, the first PR diffs against pristine
upstream and shows only our changes. The upstream version bump never appears in
it.

```text
upstream tag vX.Y.Z
      │
      ├──────────────► vX.Y.Z-upstream      (pristine; never touched)
      │                      │
      │                      └────────────► vX.Y.Z-aiven   (PR target; = upstream at first)
      │                                            ▲
      └──► work branch (patches on -upstream) ─PR──┘   diff = our patches only
```

## Why two branches

A GitHub PR shows the diff against the merge-base of its two branches. Point a
version-bump PR at an older Aiven branch and the merge-base is the previous
upstream release, so the PR drags in the whole upstream delta and reviewers
can't tell our changes from upstream's
([example](https://github.com/aiven/telegraf/pull/226)). With the pristine
`-upstream` branch as the merge-base, the PR holds our patches and nothing
else, however far upstream moved.

## Naming

Release branches are `vX.Y.Z-upstream` and `vX.Y.Z-aiven`, for example
`v1.39.1-upstream` and `v1.39.1-aiven`. Work branches are preferably prefixed
with the version `vX.Y.Z-<topic>`: `v1.39.1-port`, `v1.39.1-fix-mysql`.
Branches from before 1.39.1 used other names, such as `aiven-release-1.37.1`
or `1.30.2-aiven`.

## Rules

Never mix an upstream bump with our patches in one PR. Moving to a new upstream
version means creating new `-upstream` and `-aiven` branches from the new tag
and porting the series onto them.

One functional patch is one commit. Cherry-picking the patch commits in order
(merge commits skipped) onto a clean `-upstream` must reproduce the `-aiven`
tree. Patches are ordered by author date, oldest first, and new patches go at
the end. This form is re-established at every port. Between ports, fix commits
are allowed; they get folded into their patch at the next port.

Further rules:

- Changes that serve the fork rather than Telegraf's behaviour (docs such as
  `AIVEN_MAINTENANCE.md`, agent guidance such as `CLAUDE.md`, our CI
  workflows, repo configuration) live together in one commit, pinned as the
  first commit of the series. It is the one exception to author-date order.
  Keep these files out of plugin patches.
- Add new files instead of editing upstream-owned ones (e.g.
  `.github/workflows/aiven-*.yml`). To turn off an upstream workflow, disable
  it in the repository's Actions settings.
- Don't edit upstream files only to satisfy a linter. Such edits conflict on
  every later port. Change how our CI runs the linter instead.
- PRs are merged with a merge commit. See [Merging PRs](#merging-prs).
- The `-aiven` tip must build and pass tests after every merge. Intermediate
  commits are best-effort.

The rest of this guide calls that first commit the fork infrastructure commit.

### AIVEN_CHANGES.md

The patch that adds or changes behaviour updates `AIVEN_CHANGES.md` in the same
commit. The file has one `##` section per plugin type and one `###` section
per plugin. Inside a plugin section, each patch adds its bullets under one
dated bullet. The date is the author date of the patch commit
(`git log -1 --format=%ad --date=short <commit>`), and groups are in ascending
order:

```markdown
### Elasticsearch

* 2022-01-21
  * add cross cluster replication metrics
* 2025-02-26
  * add remote-backed store stats for OpenSearch
```

A patch only adds its own dated group, plus the headings if it is the first
entry in that section.

### Authorship and commit messages

Keep the original author. `git cherry-pick`, `git rebase` and
`git commit --amend` all preserve it. To split a bundled commit into per-patch
commits, cherry-pick the original onto a throwaway branch, trim the tree to one
patch's files and amend; then assemble the series from the trimmed commits in
order. Don't re-create commits with `--author`.

Conflict-resolution and rebase commits are not patches. Fold their hunks into
the patch each one fixes. They carry no authorship of their own.

Don't use `git cherry-pick -x`, and remove any `(cherry picked from commit ...)`
lines. The referenced hash is gone after the next port.

## Merging PRs

The repo allows only merge commits for PRs. History on `-aiven` is
semi-linear: the upstream tag, then each PR's patch commits in a straight line,
closed by that PR's merge commit.

```text
upstream tag            M1 (PR #1)            M2 (PR #2)
     o-----------------------o---------------------o   vX.Y.Z-aiven
      \                     / \                   /
       A1 ---- A2 ---- A3 -'   B1 ------------ B2-'
```

`git log --first-parent vX.Y.Z-aiven` shows one entry per PR.

- The PR branch must be up to date with its `-aiven` base before merging. The
  ruleset enforces this ("Require branches to be up to date before merging").
- Update a PR branch by rebasing, never by merging the base into it. Use
  "Update with rebase" in the PR, or locally:

  ```bash
  git fetch origin
  git rebase --no-update-refs origin/vX.Y.Z-aiven
  git push --force-with-lease
  ```

- Never resolve conflicts inside a merge commit. Rebase the PR branch and
  resolve them there.
- Merge commits carry no changes of their own. An up-to-date branch gives you
  this: the merge commit's tree equals its PR's last commit.

Patch commits keep their hashes and signatures; the merge commit records the PR
number and who merged it.

Check that every merge is clean before porting. CI runs the same check on every
push to `v*-aiven`:

```bash
for m in $(git rev-list --merges vPREV-upstream..vPREV-aiven); do
  git diff --quiet $m^2 $m || echo "merge $m carries extra changes - inspect with: git show --remerge-diff $m"
done
```

### Branch protection

- `v*-aiven`: require a PR, require branches to be up to date before merging,
  block force pushes and deletion.
- `v*-upstream`: block force pushes, deletion and updates.

## CI

Our workflows are the `.github/workflows/aiven-*.yml` files in the fork
infrastructure commit. Upstream's workflows are disabled in the repository's
Actions settings. They expect PRs into `master` and would fail or do nothing on
our branches.

- `aiven-pr-checks.yml` runs whenever a PR is opened, edited or updated. It
  fails if the base branch is not a `v*-aiven` branch, or if the PR branch
  contains merge commits.
- `aiven-build-test.yml` runs on PRs into `v*-aiven` and on pushes to those
  branches. It builds `./cmd/telegraf` and finds the Go files we changed
  relative to the matching `-upstream` branch. Those files must pass
  `gofmt -s`, and their packages, plus `./plugins`, must pass `go vet` and
  `go test -short`. On push it also runs the clean-merge check.
- `aiven-lint.yml` runs markdownlint with `.markdownlint.jsonc` on the Markdown
  files a PR into `v*-aiven` changes.

GitHub reads some configuration only from the default branch:
`dependabot.yml`, scheduled workflows and comment-triggered workflows. Putting
those in the series has no effect. PR- and push-triggered workflows run from
the branch content, so ours work on every `-aiven` branch.

## Upgrading to a new upstream version

A direct jump is fine; we went from 1.37.1 straight to 1.39.1. Smaller hops mean
fewer conflicts per PR, so choose the step size that keeps the port
reviewable.

The example below moves the series from `v1.39.1` to `v1.40.0`.

### 1. Create the two branches

```bash
git fetch upstream --tags

git switch -c v1.40.0-upstream v1.40.0
git push -u origin v1.40.0-upstream

git switch -c v1.40.0-aiven v1.40.0-upstream
git push -u origin v1.40.0-aiven
```

Creating the `-upstream` branch is pure mechanics and a good candidate for
automation.

### 2. Replay the patches onto the new upstream

Run the clean-merge check from [Merging PRs](#merging-prs) on the previous
release first. Then cut a work branch from the new pristine branch and replay
only the previous release's patch commits, skipping merge commits. Conflicts
get resolved per commit, so each fix lands in the patch it belongs to.

```bash
git switch -c v1.40.0-port v1.40.0-upstream
git cherry-pick $(git rev-list --reverse --no-merges --topo-order v1.39.1-upstream..v1.39.1-aiven)
# resolve conflicts per commit, then: git cherry-pick --continue
```

The rebase equivalent drops merge commits automatically:

```bash
git switch -c v1.40.0-port v1.39.1-aiven
git rebase --no-update-refs --onto v1.40.0-upstream v1.39.1-upstream v1.40.0-port
# resolve conflicts per commit, then: git rebase --continue
```

`v1.39.1-upstream` is the base the old patches sat on. For release lines that
predate this model, use the upstream tag they were built on instead.

Both ranges include the fork infrastructure commit, so it lands first again.
It rarely conflicts, since its files exist only in the fork.

Pass `--no-update-refs` on every rebase. If `rebase.updateRefs` is enabled in
the git config, git otherwise moves any local branch that points into the
rebased range, such as the previous `-aiven` branch.

Refine the work branch freely before opening the PR: reorder commits, fold fixes
into their patch with `git commit --fixup` and
`git rebase -i --no-update-refs --autosquash`, and force-push. Rewriting an
unmerged work branch is expected.

### 3. Verify the port branch

```bash
make tidy && make build && make test
git diff <known-good-ref> v1.40.0-port --stat
git range-diff v1.39.1-upstream..v1.39.1-aiven v1.40.0-upstream..v1.40.0-port
```

Compare against a known-good reference when one exists. For the 1.39.1 port it
was `origin/master`, which already carried upstream 1.39.1 plus our changes.
Every difference should be explainable. `git range-diff` pairs each old patch
with its new version, which shows what conflict resolution changed.

### 4. Open and merge the PR

```bash
git push -u origin v1.40.0-port
gh pr create --base v1.40.0-aiven --head v1.40.0-port \
  --title "Port Aiven patches to v1.40.0"
```

The PR diff is our series against pristine upstream: the fork infrastructure
commit, then one commit per patch. After review, merge it into `v1.40.0-aiven`
with a merge commit, then check the result:

```bash
git switch v1.40.0-aiven && git pull
git log --oneline --no-merges v1.40.0-upstream..v1.40.0-aiven
```

## Mid-version changes

Everything still goes through a PR based on `-aiven`.

A new patch is a branch off `-aiven` with one commit that also updates
`AIVEN_CHANGES.md`. The PR diff shows only that commit, and merging adds it
plus a merge commit.

A fix to an existing patch lands as a follow-up commit through a PR, merged the
same way, so it gets normal review and CI. At the next port it is folded into
the patch it belongs to (reorder plus `fixup` on the port branch). Changes to
docs, agent guidance or CI work the same way and get folded into the fork
infrastructure commit.

## Patch ordering and dependencies

Some patches build on earlier ones. The procstat swap metrics need the
`aiven-procstat` plugin, and the OpenSearch RemoteStats patch extends the
elasticsearch cross-cluster replication patch. Author-date order already puts
every follow-up after the patch it builds on, so replaying the series in order
is enough. The fork infrastructure commit stays first, outside that order; no
patch depends on it.

`git log --reverse --no-merges vX.Y.Z-upstream..vX.Y.Z-aiven` gives the order,
and `AIVEN_CHANGES.md` gives the reason for each patch.
