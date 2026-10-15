# Legacy gob messages

`legacy.gob` was encoded with Xshard commit
`7e35a67` (module `github.com/myl7/mingchain`) using Go 1.27.2, before changing
the module or its default gob registrations. A temporary test in package
`mingchain` wrote the values returned by `gobCompatibilityMessages` with one
`gob.NewEncoder(file)`, calling `Encode` on each `Msg` in order. The values use
synthetic key fields for serialization only; they are not cryptographic fixtures.

Keep this historical file unchanged when updating the current implementation.
The test decodes and checks its values rather than comparing encoded bytes.
It covers all six locally registered concrete types, including blocks nested
inside PBFT request, preprepare, and reply interfaces.

`main.go` is an independent copy of the old message schema. It does not import
the Xshard package under test. Its explicit registrations are the original
module's gob names; `[]Tx` used the package-qualified name `[]mingchain.Tx`,
while named structs used the full module path. The test sends current messages
through this decoder in a separate process and verifies the re-encoded values.
This also exercises the reverse direction without registering old and new Go
types under the same gob names in one process.

Neither test depends on matching process-global gob type IDs or test execution
order. Merely changing the module path, without retaining the wire names, makes
both directions fail with an unregistered interface type error.

`legacy-signing.json` records SHA-256 digests of the original PBFT request and
reply signing preimages from the same source revision and toolchain. They were
captured in a fresh process using the request and reply bodies from
`gobCompatibilityMessages`, in that order, with a fresh gob encoder for each.
The current preimage check also runs in a fresh test process, after the real
package initialization, so unrelated tests cannot change its gob type IDs.
Unlike the message round-trip checks, exact signing preimages matter because
peers verify the signatures over these bytes.
