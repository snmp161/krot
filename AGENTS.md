# AGENTS.md — Rules for LLM Agents Working in the krot Repository

## Core Rules

1. **`SPEC.md` is authoritative.** Tests come second, and code comes third.
   If a test or the code conflicts with `SPEC.md`, fix the test or the code, not the specification.
2. **Do not modify `SPEC.md` without an explicit decision from a human.** If you find an ambiguity or contradiction, stop, describe the problem, and propose options. Do not resolve it by making assumptions.
3. **Write tests before code.** For each task:
   1. Read the relevant sections of `SPEC.md`;
   2. Write tests based on the specification, with a reference to the applicable section (`// SPEC 7.3 V-21`);
   3. Run the tests and confirm that the new tests **fail**;
   4. Commit the tests: `test(<scope>): <what is being tested>`;
   5. Implement the code;
   6. Make all checks pass: `make fmt-check` succeeds, `go vet ./...` is clean, and `go test -race ./...` passes;
   7. Commit the code: `feat|fix|refactor(<scope>): <summary of changes>`.
4. **Create a git commit after every task.** A task is not complete until it has been committed. Do not combine changes from multiple tasks in a single commit.
5. Take tasks in order from `SPEC.md`, section 10 (T0, T1, …).

## Commits

- Use the Conventional Commits format: `type(scope): description`.
- Allowed types: `test`, `feat`, `fix`, `refactor`, `docs`, `ci`, `build`, `chore`.
- Include the task number and relevant specification sections in the commit body: `Task: T5`, `Spec: 7.3`.
- Do not commit with failing tests, except for a `test:` commit in step 3.4, where failures are expected. In that case, add `Expected: failing` to the commit body.
- Never run `git push --force` or rewrite history.

## Code

- Use the latest stable version of Go, pinned in `go.mod`.
- Use only the standard library, `golang.org/x/crypto`, and one QR code library. Adding a new dependency requires human approval.
- `CGO_ENABLED=0`.
- Favor simplicity over elegance: avoid unnecessary abstractions, frameworks, DI containers, and code generators.
- Run external commands only through the executor interface (SPEC 8.2), so they can be replaced with fakes in tests.
- Use only `crypto/rand` for randomness in production code; in tests, use a deterministic source with a seed.
- Provide operators with clear error messages and return the correct exit codes (SPEC 4.5).

## Third-party licenses

- Before adding any third-party component, identify its license, check compatibility with the project, and determine its license obligations for source and binary distribution. After approval, meet every applicable obligation, including preservation of copyright, license, patent, notice, or source materials where required.
- Keep `THIRD_PARTY_NOTICES` and `packaging/copyright` current. Include required notices in DEB/RPM packages and alongside standalone release binaries; do not assume that a permissive license removes attribution obligations.
- Review the actual packaged and published artifacts, not only dependency metadata, when adding, removing, or upgrading a third-party component.

## Documentation and Translations

- Every change to documentation or user-facing text requires native-level proofreading of each changed language before it is committed. This covers README files, `docs/`, `CONTRIBUTING.md`, `AGENTS.md`, man pages, CLI help, message catalogs in `internal/i18n`, and release notes.
- Proofread as a native speaker who specializes in software development and systems operations: use established industry terminology; keep the terms, tone, and punctuation of the existing text in that language; and preserve commands, flags, paths, placeholders, format verbs, and technical meaning.
- Use these language variants:

  | Language | Variant |
  |---|---|
  | English | United States (`en_US`) |
  | Russian | Russia (`ru_RU`) |
  | Spanish | Mexico (`es_MX`) |
  | Portuguese | Brazil (`pt_BR`) |
  | Korean | South Korea (`ko_KR`) |
  | Japanese | Japan (`ja_JP`) |
  | Chinese, Simplified | Mainland China (`zh_CN`) |
  | Chinese, Traditional | Mainland China terminology in Traditional script |
  | Arabic | Modern Standard Arabic |
  | French | France (`fr_FR`) |
  | German | Germany (`de_DE`) |

- When a text exists in several languages, proofread every translation you add or change; do not leave a translation unreviewed because the source language was reviewed.
- Add `Review: AI native-level technical proofreading` to the body of each commit that adds or changes proofread text.

## Tests

- Unit tests must not require root privileges or modify the system.
- Integration tests must use the `integration` build tag and run only inside separate network namespaces. Never modify the host network.
- Store golden files in `testdata/`. Update them only deliberately, with an explanation in the commit message.
- The requirement that operations remain invisible to clients (SPEC 6.10) must be covered by both unit tests (using a fake executor) and an integration test. These tests must not be disabled, skipped, or weakened.

## Releases

- Before creating an annotated release tag, write curated release notes describing user-visible behavior, compatibility, upgrade considerations, and security fixes when applicable. Do not substitute raw commit titles or automatically generated commit lists.
- Add or update the version entry in `packaging/changelog.yaml`, run `make changelog`, and commit both that source and the generated `CHANGELOG.md` before tagging. The entry version and UTC date must match the intended tag.
- DEB/RPM changelogs and GitHub release notes must be generated from the same curated entry. A missing, empty, stale, or mismatched entry must stop the release.
- Verify artifact attestations before publication as required by SPEC 9.2. Never move or delete a published release tag.

## Prohibited Actions

- Modifying `SPEC.md` on your own initiative.
- Deleting, disabling (`t.Skip`), or weakening tests to make the code pass.
- Using `wg-quick down/up`, `systemctl restart`, or recreating an interface in client operations.
- Storing clients' private keys anywhere.
- Writing installation instructions or scripts that use `curl … | bash`.
