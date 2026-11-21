# The architecture of auto8

This document holds the architecture and the tech stack of auto8. Every plan in `docs/plans` and every build follows it. The operator owns it as a process file, so a change merges only after the operator approves it.

The first source of these decisions is the Planning Contract of `docs/plans/2026-11-20-2358-feat-auto8-automatic-hunter-plan.md`. Its key technical decisions (KTD) keep their numbers here in brackets.

## Authority

A plan can add detail to this document, but it cannot contradict it. If a plan needs a different decision, the plan names it in a key technical decision with the words "changes `docs/ARCHITECTURE.md`". The operator approves the plan, and the first pull request of that plan changes this document.

The build agent never changes this document. If a unit does not fit this document, the build stops and the issue goes to `blocked` with the reason.

## Tech stack

auto8 is written in Go. The version is the one in `go.mod`, today Go 1.27.1. The module path is `github.com/opsotter/auto8`.

The standard library comes first. auto8 uses these third-party modules:

| Module | Use |
|---|---|
| `github.com/eclipse/paho.mqtt.golang` v1.5.1 | The MQTT client (KTD6) |
| `github.com/BurntSushi/toml` v1.6.0 | The configuration file (KTD6) |
| StationA `shared/mqtt`, `shared/schema`, `shared/logging` | Vendored into `third_party/stationa-shared/` (KTD6) |

A new third-party module needs a key technical decision in an approved plan. The decision names the module, its version, and the reason that the standard library is not sufficient. auto8 never imports opsotter code.

auto8 carries its own codec for the WSJT-X UDP protocol, schema 3, with the message types of ws-hunter (KTD3, KTD4). It does not use a third-party WSJT-X library.

## Process model

auto8 is one process, which `cmd/auto8` starts (KTD1). It has no database server and no second process. It keeps its small local lists in a file in the user state directory, outside the repository (KTD16).

The state of the TX gate has one owner goroutine. Other goroutines send it requests and inputs over channels (KTD1).

## Packages and imports

The product code lives in these packages (KTD1):

| Package | Role |
|---|---|
| `cmd/auto8` | Starts the process, builds the send handles, wires the packages |
| `internal/config` | Reads and checks the TOML configuration |
| `internal/wsjtx` | Receives and decodes WSJT-X datagrams |
| `internal/bus` | MQTT subscriptions and typed publishes |
| `internal/hunt` | Needs, task choice and the local lists |
| `internal/bands` | Band foraging, mode choice and TX offset choice |
| `internal/ui` | The read-only state page |
| `internal/tx` | TX control: the gate, with `send/` for the senders and `web/` for Enable and Stop |

Imports go in one direction. `internal/tx` imports only the types of `wsjtx`, `bus` and `config`. `hunt`, `bands` and `ui` import only the request and state types of `tx`. No package outside `internal/tx` imports `internal/tx/send`.

The process tool of the development life cycle lives in `cmd/auto8-flow` and `internal/flow`. It is a process file. The product code never imports it, and it never imports the product code.

## TX control boundary

TX control is any code that can make WSJT-X or MSHV transmit. It is also the code that decides the time at which auto8 can ask for a transmission. All TX control code lives in `internal/tx` (KTD2).

Only `cmd/auto8` builds the UDP send handle and the raw publish handle for `radio/cmd`. It gives them only to the gate. `internal/bus` exports typed publish methods for the auto8 topics, and no general publish. If code outside `internal/tx` can send a TX command, a test in `internal/tx` (`boundary_test.go`) fails.

## Interfaces to other systems

auto8 talks to other systems only through these interfaces:

| Peer | Interface | Direction |
|---|---|---|
| WSJT-X or ws-hunter | UDP, WSJT-X protocol schema 3, fixed peer address (KTD4) | Both |
| StationA and chase-queue | MQTT, by the StationA bus contract (KTD6, KTD17, KTD18) | Both |
| horstreporter | HTTP, read only (KTD13) | In |
| The operator | A local web page on a loopback address (KTD5) | Both |

A change to a topic or a payload that another project reads is a change in that project (`CLAUDE.md`).

## Configuration and secrets

The configuration is one TOML file. `config.example.toml` in the repository shows every key with invented values. Each limit has a minimum and a maximum, and auto8 refuses a value outside them (KTD6, KTD15).

Secrets come only from the environment, for example `AUTO8_MQTT_PASSWORD`. auto8 refuses a password in the file. No secret goes into a log or an MQTT payload.

## Tests

Every package has unit tests that run with `go test -race ./...`. The integration tests live in `test/integration` behind a build tag. They run auto8 against a simulated WSJT-X, a real Mosquitto broker in a container, and a simulated chase-queue.

Test data holds only invented callsigns and grids, and no host addresses. A test rejects any call in test data that is not on the list of invented calls (KTD21). No captured datagram goes into the repository.

## What exists today

On 2026-11-21 the repository has the process tool (`cmd/auto8-flow`, `internal/flow`) and a stub in `cmd/auto8`. The product packages in the table above do not exist yet. The pull requests of the hunter plan add them in the order of its Implementation Units.
