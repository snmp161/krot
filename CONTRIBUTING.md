# Contributing to krot

Thank you for improving krot. This guide applies to code, tests, documentation,
translations, packaging, and CI/CD changes.

By participating, you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).
Report vulnerabilities privately as described in [SECURITY.md](SECURITY.md), not
in public issues or pull requests.

## Before you start

- Read [README.md](README.md) for the user-facing behavior and
  [docs/SPEC.md](docs/SPEC.md) for the authoritative technical requirements.
- Check existing issues, pull requests, and [docs/BACKLOG.md](docs/BACKLOG.md).
- For a feature or behavioral change, agree on the task and specification with
  the maintainer before implementation. Do not change `docs/SPEC.md` on your own
  initiative. If it is ambiguous or contradicts tests or code, stop and propose
  options instead of guessing.
- Keep tasks small and ordered. A user-visible feature, a documentation update,
  and unrelated cleanup should not share a commit.

## Development environment

Use the Go version pinned in `go.mod`. The same toolchain selection must work for
local development and CI. Production builds are static and use
`CGO_ENABLED=0`.

Unit tests must not require root or modify the host system. Integration tests
require Docker and run in a privileged container with isolated network
namespaces. They may load WireGuard kernel modules through the shared host
kernel; review `scripts/integration-docker.sh` before running them.

No installation command in project documentation may use `curl … | bash` or an
equivalent pipe to a shell.

## Test-driven workflow

1. Read the relevant specification sections.
2. Add tests based on those requirements. Reference the section in the test name
   or a comment, for example `// SPEC 7.3 V-21`.
3. Run the new tests and confirm that they fail for the expected reason.
4. Commit the tests separately:

   ```sh
   git commit -m "test(<scope>): <behavior>" \
     -m "Task: T<number>" -m "Spec: <sections>" -m "Expected: failing"
   ```

5. Implement the change without deleting, skipping, or weakening existing tests.
6. Run all required checks and review the final diff.
7. Commit the implementation with `feat`, `fix`, or `refactor` as appropriate.

Use [Conventional Commits](https://www.conventionalcommits.org/) with an allowed
type: `test`, `feat`, `fix`, `refactor`, `docs`, `ci`, `build`, or `chore`.
Include the task number and relevant specification sections in the commit body.
Except for the test commit in step 4, never commit with failing tests. A task is
complete only when it is committed; do not combine several tasks in one commit.
Do not rewrite published history or force-push.

## Required checks

Before submitting a change, run:

```sh
make fmt-check
go vet ./...
go vet -tags integration ./...
go mod tidy -diff
go mod verify
go test -race -count=1 -shuffle=on ./...
```

`make check` runs the complete standard suite, including formatting, vet,
module hygiene, shuffled race-enabled unit tests, gosec, staticcheck, ShellCheck,
and actionlint. Audit tools and the ShellCheck image are pinned by the repository.
Formatting must be checked with the Go toolchain selected from `go.mod`.

The underlying unit command is `go test -race ./...`; CI and the Makefile target
add `-count=1 -shuffle=on`. Run `make fuzz-smoke` for bounded fuzzing of every
existing target and `make package-preflight` to build DEB/RPM packages for both
architectures, run Lintian/rpmlint and apt/dnf lifecycle checks, and execute
representative smoke tests on Ubuntu 24.04 and AlmaLinux 9.

The unit suite must maintain the coverage threshold enforced by CI. Add golden
files under the relevant package's `testdata/` directory. Update golden files
only deliberately, review every difference, and explain the reason in the commit
message.

For package or release changes, build the affected DEB/RPM artifacts and run the
corresponding `scripts/smoke.sh` scenarios when Docker is available.

## Integration tests

Run the full integration suite with:

```sh
make integration
```

Never run integration tests directly on the host. They require the `integration`
build tag and isolated network namespaces. The requirement that client operations
remain invisible to connected clients (SPEC 6.10) must be covered both by unit
tests with the fake executor and by an integration test that proves existing
clients keep their interface, handshake state, and traffic. These tests must not
be disabled, skipped, or weakened.

## Code and dependencies

- Favor simple, readable standard-library code over frameworks, generators, or
  unnecessary abstractions.
- Production randomness must use `crypto/rand`; tests use deterministic seeded
  sources.
- Run external commands through the executor interface so unit tests can use a
  fake runner.
- Do not store clients' private keys. Never commit private keys, credentials, or
  sensitive configuration data.
- Do not use `wg-quick down/up`, `systemctl restart`, or interface recreation for
  client operations.
- Only the standard library, `golang.org/x/crypto`, and the existing QR library
  are permitted. Adding another dependency requires explicit maintainer approval.
- Keep clear operator-facing errors and the exit codes defined in SPEC 4.5.

## Third-party licenses

- Before adding a third-party component, identify its license, confirm that it
  is compatible with the project, and determine its obligations for source and
  binary distribution. After approval, meet every applicable obligation,
  including preservation of copyright, license, patent, notice, or source
  materials where required.
- Keep `THIRD_PARTY_NOTICES` and `packaging/copyright` current. Include the
  required notices in DEB/RPM packages and alongside standalone release binaries;
  a permissive license does not remove attribution obligations.
- When adding, removing, or upgrading a third-party component, review the actual
  packaged and published artifacts, not only dependency metadata.

## Documentation and localization

Keep README files, CLI help, man pages, examples, and the specification
consistent with the implementation. Do not translate command names, flags, JSON
fields or values, configuration keys, KeeneticOS commands, shell completion, or
the UDP protocol.

Application and man-page localization covers the same languages as the project
README files. Every change to documentation or user-facing text requires
native-level proofreading of each changed language before it is committed. This
covers README files, `docs/`, `CONTRIBUTING.md`, `AGENTS.md`, man pages, CLI
help, message catalogs in `internal/i18n`, and release notes. Native-level AI
editorial review is required; review by a human native speaker is also welcome.

Proofread as a native speaker who specializes in
software development and systems operations: use established industry
terminology; keep the terms, tone, and punctuation of the existing text in that
language; and preserve commands, flags, paths, placeholders, format verbs, links,
formatting, and technical meaning. Use these language variants:

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

When a text exists in several languages, proofread every translation you add or
change; a reviewed source text does not cover its translations. Run structural
checks for every localized variant, and add
`Review: AI native-level technical proofreading` to the body of each commit that
adds or changes proofread text.

Update installation examples to the current release version. Do not add
instructions that bypass checksum verification.

## Pull requests

A pull request should:

- describe the problem, the chosen behavior, and the specification sections;
- contain the tests that demonstrate the behavior;
- keep unrelated changes out;
- pass all required CI and integration jobs;
- update documentation and translations affected by the change;
- explain any golden-file, packaging, dependency, or CI changes;
- avoid private keys and sensitive logs in the description and attachments.

Review the CI results and respond to review comments with follow-up commits.
Maintainers may ask for a different scope or approach when the specification is
unclear.

## Releases

Releases are created by maintainers from an annotated `v*` tag. The tag must point
to the intended commit on `main`, and the complete release workflow must pass
before artifacts are published. Never move or delete a tag after its release has
been published.

Before tagging, write curated release notes in `packaging/changelog.yaml` and run
`make changelog` to regenerate `CHANGELOG.md`. Describe behavior, compatibility,
upgrade considerations, and security fixes when applicable; do not substitute raw
commit titles or an automatically generated commit list. Commit both files before
tagging; the entry version and its UTC date must match the intended tag. The same
entry feeds the GitHub release notes and the DEB/RPM package changelogs. A
missing, empty, stale, or mismatched entry stops the release. Verify all required
artifact attestations before publication.

## License

Contributions are submitted under the project's [Apache-2.0](LICENSE) license.
