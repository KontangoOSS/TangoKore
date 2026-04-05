# TangoKore

> **You are your password. Keep being you.**

TangoKore is the easiest way to connect your machines securely. No passwords. No complicated setup. Just one command.

---

## 🚀 Get Started in 30 Seconds

```bash
# That's it. Just run this one command:
curl https://ctrl.konoss.org/install | sudo sh
```

Your machine is now on a secure network. Connected to everything it needs. Authenticated by being itself.

---

## What Does That Command Do?

1. **Downloads** the installer from our server
2. **Runs** the installer (it asks for your password once)
3. **Connects** your machine to the secure mesh network
4. **Done!** Your machine can now talk to other machines securely

That's all you need to know. The rest happens automatically.

---

## What Makes It Different?

### No Passwords

Your machine proves who it is by **being itself**. Its hardware, OS, behavior—that's your identity. You can't forget it. You can't lose it. It's impossible to fake.

### No Setup

No config files. No firewall rules. No IP addresses to memorize. Just one command.

### No Hidden Stuff

We don't collect anything secret. We collect hardware info (like what `uname -a` shows). It's public. You control who sees it.

- **Using our server?** We keep your data safe, encrypted, deletable anytime
- **Running your own?** Your data never leaves your network

### Fully Open Source

All code is public. MIT license. Read it. Audit it. Trust it.

---

## How It Works (Simple Version)

**First time you run it:**
- Your machine tells us what hardware it has
- We give it read-only access (safe default)
- You prove you're trustworthy over time

**Second time you run it:**
- We recognize your machine instantly
- We restore your previous permissions
- Everything just works

**With credentials:**
- You skip the waiting and get full access immediately

---

## Your Privacy Is Important

### You Control Your Data

**Option 1: Use Our Server**
- Fast, simple, hosted by Kontango
- Your data is encrypted, secure, deletable
- GDPR/CCPA compliant
- Email privacy@konoss.org anytime to delete

**Option 2: Run Your Own**
- Your machine's data never leaves your network
- You control everything
- 100% private

### No Tracking

We don't sell your data. We don't track you. We don't analyze your usage. Period.

---

## Real Examples

### Scenario 1: Your First Machine

```bash
curl https://ctrl.konoss.org/install | sudo sh
```

Your machine enrolls with basic read-only access. Safe default.

### Scenario 2: Your Machine Comes Back

You deleted the certificate but your hardware is the same.

```bash
curl https://ctrl.konoss.org/install | sudo sh
```

We recognize your hardware. You get your permissions back. No re-setup.

### Scenario 3: Pre-Provisioned Access

Your admin pre-gave you credentials.

```bash
curl https://ctrl.konoss.org/install | sudo sh --role-id xxx --secret-id yyy
```

You skip quarantine. You get full access immediately.

---

## When Do You Need to Think?

**You don't.** That's the whole point.

The one time you interact is when you run the command. After that, everything is automatic. You don't remember passwords. You don't configure anything. Your machine just works.

---

## What Next?

### For Casual Users

That's it. You're done. Your machine is secure and connected.

### For Advanced Users

Want to understand more? Check out the **[detailed docs](https://docs.konoss.org)** where we explain:
- How fingerprinting actually works
- How the trust system works
- How to deploy on your own server
- Advanced configuration

### For Developers

The source code is on GitHub. Everything is open. Build on top of it. Modify it. Audit it.

```
github.com/KontangoOSS/TangoKore
```

---

## But What About...?

**"Is my data really safe?"**

Yes. Your data is encrypted with AES-256. You control who sees it. If you run your own server, only you see it.

**"What if I lose my machine?"**

Back up your machine file (`~/.kontango/machine.json`) and you can restore from any machine instantly.

**"What if I forget my password?"**

You don't have one. Your machine is your password. You can't forget hardware.

**"Is this open source?"**

Yes. MIT license. [Read the code.](https://github.com/KontangoOSS/TangoKore)

---

## Privacy Notice

When you run the installer, your machine sends:

- **Operating system** (what you use: Linux, macOS, Windows)
- **Hardware info** (CPU, memory, network interfaces)
- **System identifiers** (things like `uname -a` shows publicly)

This is **public hardware information**. Not passwords. Not secrets. Just hardware details.

Why? So we can recognize your machine again and restore your settings.

**You can always:**
- See what's being sent (it's logged)
- Delete your data from our system (email privacy@konoss.org)
- Run your own server (data stays private)
- Opt-out (don't run the installer)

---

## License

MIT License. Use it anywhere. Modify it. Make it better.

---

## Questions?

- **Privacy:** privacy@konoss.org
- **Technical:** GitHub Issues
- **General:** konoss.org

---

**One command. Your machine is secure.**

```bash
curl https://ctrl.konoss.org/install | sudo sh
```

That's all you need.
