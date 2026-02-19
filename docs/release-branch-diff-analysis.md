# Release Branch Diff Discrepancy: Root Cause Analysis

## Problem

The diff between `release/v3.1` and `release/v3.2` appears much larger than it
actually is. Although many commits share the same patch content (same PR, same
code change), they have different SHA hashes on each branch. Standard Git
commands like `git log` and `git diff` treat them as distinct commits, inflating
the computed change list.

## Root Cause

The issue is caused by a **merge commit** on `release/v3.1` that broke the
linear shared history between the two branches.

### What happened

1. `release/v3.1` was **not** branched directly from `main`. Instead, an older
   branch (`c47821f9`, at `powens` PR #544) was **merged** with `main`:

   ```
   8303d3fa Merge main into release/v3.1 to restore proper commit history
   Merge: c47821f9 c607fa3a
   ```

   Since `c47821f9` was already an ancestor of `c607fa3a`, this merge could have
   been a fast-forward but was recorded as a merge commit instead.

2. After the merge commit, subsequent fixes (#618, #619, #622, #623, #625, #631,
   #634, #635, #636, #637, #638) were **cherry-picked** onto `release/v3.1`.

3. `release/v3.2` was branched directly from `main` (currently identical to
   `main` at `a31f7742`), so it has the **original** commits with their original
   hashes.

### Why the hashes differ

A Git commit's SHA hash is computed from its content **and its parent hash**.
Cherry-picked commits produce the same *patch* (same code diff) but have a
different *parent* (the merge commit `8303d3fa` instead of the original parent
on `main`). Therefore:

| PR   | Hash on `release/v3.1` | Hash on `release/v3.2` | Same patch? |
|------|------------------------|------------------------|-------------|
| #618 | `6e947aa7`             | `73d11418`             | Yes         |
| #619 | `6c7b3bf6`             | `cfea7773`             | Yes         |
| #623 | `e14727c2`             | `d0ab3608`             | Yes         |
| #625 | `3836717c`             | `46f2d51e`             | Yes         |
| #622 | `6354a24b`             | `cecf6de6`             | Yes         |
| #631 | `6ef79a92`             | `949c391b`             | Yes         |
| #634 | `10cf4baa`             | `abef5b25`             | Yes         |
| #635 | `e4c3280d`             | `240e8dd5`             | Yes         |
| #636 | `a2836343`             | `abff3270`             | Yes         |
| #637 | `d6f1d32d`             | `c27702e7`             | Yes         |
| #638 | `14db9622`             | `c508710d`             | Yes         |

Git sees 11 "different" commits on each side even though they carry identical
patches.

### Visual summary

```
main ─── ... ─── c607fa3a ─── 73d11418(#618) ─── cfea7773(#619) ─── ... ─── a31f7742
                     │                                                          ↑
                     │                                                    release/v3.2
                     │
                     ├─── (merge) 8303d3fa ─── 6e947aa7(#618') ─── ... ─── 14db9622
                     │         ↑                                                ↑
                  c47821f9     │                                          release/v3.1
                          (old v3.1)
```

## Actual diff

Using `git log --cherry-pick` (which compares patch-ids to skip equivalent
commits), the **real** difference between the branches is:

**Only in `release/v3.2`** (5 commits — the actual new features):
- `a31f7742` feat: add Coinbase Prime connector plugin (#643)
- `d2ba4ceb` feat: payout capability for generic connector (#632)
- `6f4150a5` feat: connector workbench (#641)
- `1409b6c0` fix: use TransferPeer.Type to classify Fireblocks payment types (#642)
- `766bacbb` feat: Add Fireblocks connector plugin (read-only) (#639)

**Only in `release/v3.1`** (2 commits — release bookkeeping):
- `2636348a` release v3.1.0
- `8303d3fa` Merge main into release/v3.1 to restore proper commit history

## Recommendations

### Immediate fix: use `--cherry-pick` for change lists

When computing the changelog between release branches, use:

```bash
# Commits in v3.2 that have no patch-equivalent in v3.1
git log --cherry-pick --right-only --oneline release/v3.1...release/v3.2

# Commits in v3.1 that have no patch-equivalent in v3.2
git log --cherry-pick --left-only --oneline release/v3.1...release/v3.2
```

The `--cherry-pick` flag compares commits by **patch-id** (content hash of the
diff) rather than commit SHA, correctly filtering out cherry-picked duplicates.

For file-level diffs, `git diff release/v3.1..release/v3.2` already works
correctly since it compares trees, not commit lists.

### Future releases: branch from main, don't merge into release branches

The reason `release/v3.2` doesn't have this problem is that it was created by
branching directly from `main`. This means commits on `release/v3.2` share the
exact same SHA hashes as `main`.

The recommended workflow for creating a new release branch:

```bash
# Good: branch from the desired point on main
git checkout -b release/v3.X <commit-on-main>

# Bad: maintain a separate branch and merge main into it
git checkout release/v3.X
git merge main   # creates divergent history
```

### If backporting fixes to older release branches is needed

When a fix merged to `main` also needs to go to an older release branch:

1. **Cherry-pick is fine** for individual commits — just be aware that
   changelog tooling needs `--cherry-pick` to deduplicate.

2. **Prefer using the PR number** (e.g., `#638`) rather than commit SHA when
   referencing changes across branches in release notes, since the SHA will
   differ.

3. **Do not merge `main` into a release branch.** This creates a merge commit
   that forces all subsequent cherry-picks to have new parent chains, making the
   divergence permanent. If the release branch needs to catch up to main,
   consider creating a new release branch from main instead.
