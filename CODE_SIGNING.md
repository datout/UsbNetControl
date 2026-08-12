# Code signing policy

Free code signing provided by SignPath.io, certificate by SignPath Foundation.

This policy applies to official Windows binaries published by the `datout/UsbNetControl` project.

## Project and source

- Project: **UsbNetControl**
- Source repository: `https://github.com/datout/UsbNetControl`
- License: MIT
- Official release artifacts are built from this repository by GitHub Actions.
- Only artifacts built from source code and build scripts maintained in this repository may be submitted for signing.

## Team roles

For this single-maintainer project, the current roles are:

- **Authors / Committers:** [@datout](https://github.com/datout)
- **Reviewers:** [@datout](https://github.com/datout)
- **Approvers:** [@datout](https://github.com/datout)

Contributions from other people are accepted through pull requests and are reviewed before merge.

## Release and signing rules

1. Release binaries are built by GitHub Actions on GitHub-hosted runners.
2. The source revision, workflow and build scripts used for a release are stored in this repository.
3. Unsigned build artifacts are uploaded to GitHub Actions before a SignPath signing request is created, allowing SignPath origin verification.
4. Every release signing request requires manual approval by an Approver.
5. Only UsbNetControl binaries built from this repository are signed with the project's SignPath policy.
6. Signed PE files must contain `ProductName=UsbNetControl` and a `ProductVersion` matching the release version. The SignPath artifact configuration enforces these values.
7. Authenticode signing is performed by SignPath; certificate private keys are not stored in this repository or in GitHub Actions secrets.

## Privacy policy

See [PRIVACY.md](PRIVACY.md).

**This program will not transfer any information to other networked systems unless specifically requested by the user or the person installing or operating it.**

UsbNetControl does not contain telemetry, analytics, advertising, update checking or background network communication.

## System changes

UsbNetControl is an administrative utility and intentionally changes local Windows configuration only after an explicit user action in the graphical interface. It can:

- change Windows removable-storage access policy;
- enable or disable a selected local network adapter.

The application requests administrator privileges at startup. The interface describes the requested action, and disabling the active/default network adapter is presented as a disruptive action because it can immediately disconnect the computer from the network.

## Verification

After SignPath signing is enabled, official release executables can be checked in Windows with:

```powershell
Get-AuthenticodeSignature .\UsbNetControl-vX.Y.Z-win-x64.exe | Format-List
```

The expected signer for Foundation-provided signing is **SignPath Foundation**.
