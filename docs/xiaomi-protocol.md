# Xiaomi miio Protocol Reference

Background notes on the wire protocol used by [backend/mi/xiaomi.go](../backend/mi/xiaomi.go) to talk to Xiaomi Chuangmi smart plugs (model M1) over UDP.

## Overview

Commands are sent over **UDP port 54321** and protected with **AES-128-CBC**. The encryption key and IV are both derived from a per-device 16-byte token.

## 1. Discovery (hello handshake)

Before sending commands, the driver must learn two things from the device:

- **Device ID** — 4-byte unique identifier
- **Stamp** — 4-byte counter the device expects on the next command

The handshake:

```
1. Send a "hello" packet to <device-ip>:54321 (UDP)
   - 32 bytes: magic 0x2131, length 0x0020, then 28 bytes of 0xFF padding

2. Device responds with 32 bytes:
   - bytes  8-11: Device ID
   - bytes 12-15: Current stamp value
   - bytes 16-31: Device token echo (verification)
```

## 2. Packet structure

Every command packet uses this layout:

| Offset | Length | Description                          |
|--------|--------|--------------------------------------|
| 0-1    | 2      | Magic number (`0x2131`)              |
| 2-3    | 2      | Total packet length (big-endian)     |
| 4-7    | 4      | Reserved (`0x00000000`)              |
| 8-11   | 4      | Device ID (from discovery)           |
| 12-15  | 4      | Stamp                                |
| 16-31  | 16     | MD5 checksum                         |
| 32+    | var    | AES-CBC encrypted JSON payload       |

**Checksum** = `MD5(header[0:16] || token || encrypted_payload)`

## 3. Encryption

```
key = MD5(token)              # 16 bytes
iv  = MD5(key || token)       # 16 bytes
```

The JSON command is PKCS7-padded to a 16-byte boundary, then AES-128-CBC encrypted with the derived key and IV. The same `key`/`iv` pair decrypts the response.

## 4. Stamp counter

The stamp is the protocol's replay-attack defence:

- The first command after discovery **must** echo the stamp received in the hello response.
- Each subsequent command increments the stamp by 1 (4-byte big-endian).
- A wrong stamp is silently dropped by the device.

This driver re-runs the hello handshake before every command (see `discoverDevice` in [backend/mi/xiaomi.go](../backend/mi/xiaomi.go)), so it always uses a fresh stamp and never has to track the counter itself.

## 5. Commands

Payloads are JSON. The driver currently uses two methods:

**Set power state** (used by `setPower`):
```json
{"id":1,"method":"set_power","params":["on"]}
{"id":1,"method":"set_power","params":["off"]}
```

**Query power state** (used by `miQueryPower`):
```json
{"id":1,"method":"get_prop","params":["power"]}
```

**Successful response** (also encrypted):
```json
{"result":["ok"],"id":1}      // for set_power
{"result":["on"],"id":1}      // for get_prop
```

**Error response**:
```json
{"error":{"code":-10000,"message":"..."},"id":1}
```

`setPower` validates both forms — a missing `result:["ok"]` or any `error` object is treated as a failure rather than silently caching the wrong state.

## Token acquisition

The 32-character hex token is per-device and required for encryption. Easiest path:

- **`python-miio`** — `mirobo discover` after running the device through a fresh Mi Home pairing flow will surface the token. See [github.com/rytilahti/python-miio](https://github.com/rytilahti/python-miio).
- **Packet capture during pairing** — the token appears in bytes 16-31 of the hello response on first setup.

## Troubleshooting

| Symptom                            | Likely cause                                                    |
|------------------------------------|------------------------------------------------------------------|
| Discovery times out                | Device unreachable, on different VLAN, or UDP/54321 blocked     |
| Discovery works, set_power silent  | Wrong token (decrypts to garbage, device drops the packet)      |
| `device error -10000`              | Firmware rejected the method — try `set_power` vs `toggle`      |
| First command after idle is slow   | Device wakes from low-power state; second command is normal     |

## References

- Protocol: miio (Xiaomi smart-home family)
- Model tested: Chuangmi Plug M1 (`chuangmi.plug.m1`)
- Related: [python-miio](https://github.com/rytilahti/python-miio), [homebridge-miio](https://github.com/aholstenson/miio)
