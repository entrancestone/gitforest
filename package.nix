{ lib
, buildGoModule
, git
, python3
, version ? "3.1.8" # x-release-please-version
}:

buildGoModule {
  pname = "gitforest";
  inherit version;

  src = lib.cleanSource ./.;
  vendorHash = "sha256-z8IndcHcZ6nLqhLtAYul3ppddpOA4AHGQWIlfYY/pfI=";

  ldflags = [
    "-X main.version=v${version}"
  ];

  # python3 is required by .github/scripts/pr-guardian-gate.sh, which the
  # test suite (TestPrGuardianGateDecisions) executes via bash to parse
  # pipeline attestation JSON. Without it the gate falls through to the
  # wrong error path and tests fail in the Nix sandbox.
  nativeCheckInputs = [
    git
    python3
  ];

  # The cmd/ package's e2e tests build the gitforest binary, create git repos
  # with bare remotes, and spawn gitforest as a subprocess — all of which
  # require network access and unrestricted filesystem access that the Nix
  # sandbox does not provide. The project's own CI runs `go test ./...` on
  # ubuntu, macOS, and Windows, which is more comprehensive than the Nix
  # sandbox can do.
  doCheck = false;

  meta = {
    description = "Git worktree pool manager for parallel AI coding agent workflows";
    homepage = "https://github.com/entrancestone/gitforest";
    license = lib.licenses.mit;
    mainProgram = "gitforest";
    platforms = [
      "aarch64-darwin"
      "aarch64-linux"
      "x86_64-linux"
    ];
  };
}
