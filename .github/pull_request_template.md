## Summary

<!-- Describe the problem, the chosen behavior, and the user-visible result. -->

## Task and specification

Task: T<!-- number -->
Spec: <!-- sections of docs/SPEC.md -->

- [ ] The behavior is covered by an approved section of `docs/SPEC.md`; any ambiguity was resolved with the maintainer before implementation.
- [ ] Tests were added before the implementation and reference the applicable SPEC sections.

## Verification

- [ ] `make check` passes.
- [ ] Affected integration scenarios pass with `make integration` (required for tunnel/client/network changes).
- [ ] `make package-preflight` passes for packaging or release changes.
- [ ] Coverage and all required CI gates remain green; no tests were deleted, skipped, or weakened.

## Safety and documentation

- [ ] The diff contains no private keys, credentials, sensitive configurations, or unsafe host changes.
- [ ] Affected README files, CLI help, man pages, and translations are consistent with the implementation.
- [ ] All golden files, dependencies, packaging, and CI changes are explained in the commit messages and reviewed deliberately.

## Notes for reviewers

<!-- Mention rollout, compatibility, known limitations, or areas needing particular attention. -->
