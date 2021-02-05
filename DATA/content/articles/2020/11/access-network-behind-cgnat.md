---
publishDate: "2020-11-30T09:00:00.000+01:00"
tags:
- mondayletter
- network
title: Access a network behind a Carrier Grade NAT with Wireguard
---

I recently moved from the place where I was staying at to my own studio. In addition, since I was building a computer, I wanted to be able to access remotely to its capabitities, as well as any other device I have at home. Thus, I thought: let's set up a VPN!

The problem arised when I went to the configuration of my router and saw that my WAN IP was in the format `172.xxx.xxx.xxx`. I immediately knew that that was a [private IP](https://en.wikipedia.org/wiki/Private_network). On my ignorance, I decided to search why I didn't have an actual public IP.

Well, apparently there's a thing called Carrier Grade NATs (CGNATs) which are gigantic NATs that Internet providers decided to create in order to band aid the shortage of IPv4s in many places. So I thought, once again, "let's see if I have an IPv6". I open [test-ipv6.com](https://test-ipv6.com) and for my surprise (or not) I see a score of 0 out of 10. No public IP for me.

Since the contract with the ISP is not mine, there's not much I can do. I already contacted the agency where I'm renting the studio. They usually reply quickly, but I haven't heard from them about this for a few days already. Pretty sure it's an unusual request. In the meanwhile, I decided to go for a different strategy.

The new strategy is to have a VPS with a public IP *somewhere* where I can have a Wireguard VPN running. Then, in my home network, I can have a Raspberry Pi connected to that same VPN which exposes my local network. Sounds good? Let's see how we can do that!

## Pre-requisites

You will need the following to successfully follow this guide:

- A Linux host with a public IP. I will be using a VPS.
- A Linux host inside your network. I will be using a Raspberry Pi.
- The device you want to connect to your home network while you're away.

I am not covering installation steps. The [official documentation](https://www.wireguard.com/install/) explains how to install Wireguard and its tools on most OSes. For the Raspberry Pi, please follow this [amazing guide](https://www.sigmdel.ca/michel/ha/wireguard/wireguard_02_en.html#installing_wg_raspbian).

## Key generation

First of all, we need to generate the public and private keys of all devices that will be connected in this network. For that, we can use the following sequence of commands:

```
wg genkey | tee vps_privatekey | wg pubkey > vps_publickey
wg genkey | tee pi_privatekey | wg pubkey > pi_publickey
wg genkey | tee client_privatekey | wg pubkey > client_publickey
```

## The public host (aka VPS)

Configuration file ```/etc/wireguard/wg0.conf```:

```conf
[Interface]
Address = 10.200.200.1/24
PrivateKey = <vps_privatekey>
ListenPort = 37466                  

# Do not forget to change eth0 to the interface that is connected to the Internet!
PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -A FORWARD -o %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT; iptables -D FORWARD -o %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

# This is the peer inside the local subnet we want to have access to.
# For that to happen, you need to add your local subnet to the AllowedIPs
# option just like I did below.
[Peer]
PublicKey = <pi_publickey>
AllowedIPs = 10.200.200.2/32, 192.168.1.0/24

# A client...
[Peer]
PublicKey = <client_publickey>
AllowedIPs = 10.200.200.3/32

# An additional client...
[Peer]
PublicKey = <some other public key>
AllowedIPs = 10.200.200.4/32
```

Now, you need to setup IP forwarding on the machine (you may need to also uncomment the line on `/etc/sysctl.conf` so it persists across reboots):

```shell
$ sysctl net.ipv4.ip_forward=1
```

And start the Wireguard server:

```shell
$ wg-quick up wg0
```

You can also turn it on on boot:

```shell
$ sudo systemctl enable wg-quick@wg0
```

To check if everything looks fine:

```shell
$ sudo wg show
```

## The local host (aka the Raspberry Pi)

For the host in the local network, we will create a file ```/etc/wireguard/wg0-client.conf``` with the following configuration:

```conf
[Interface]
Address = 10.200.200.2/24                  
PrivateKey = <pi_privatekey>
DNS = 1.1.1.1                 # you can also use your own DNS server

[Peer]
PublicKey = <vps_publickey>
Endpoint = <vps ip>:37466     # do not forget to put your VPS IP here
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25      # keep connections alive across NAT
```

You have to add two `iptables` rules permanently. Since the command is temporary and does not persist reboots, I recommend adding them to the file `/etc/rc.local`.

```shell
$ iptables -A FORWARD -i wg0-client -j ACCEPT
$ iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
```

Similarly to what we did on the public host, we need to enable IP forwarding. Do not forget to make it permanent.

```shell
$ sysctl net.ipv4.ip_forward=1
```

Set up the Wireguard client:

```shell
$ wg-quick up wg0-client
```

And enable it on boot:

```shell
$ sudo systemctl enable wg-quick@wg0-client
```

So far, from the VPS, you should be able to ping any device on your local network. If that works, then the setup is working and we now only need to setup the third client, the client that you will connect to the Wireguard when you're outside your home and need to access some of your devices.

## The client

The client (your smartphone, computer, etc) is the easiest. Here's the configuration:

```conf
[Interface]
PrivateKey = <client_privatekey>
Address = 10.200.200.3/24
DNS = 1.1.1.1                     # you can use your own!

[Peer]
PublicKey = <vps_publickey>
AllowedIPs = 0.0.0.0/0
Endpoint = <vps ip>:37466
PersistentKeepalive = 25
```

On your phone, it can be easily added through the Wireguard app. The same on the computer. This will create a **full tunnel** VPN. However, if you just want to access your local network, while using your current Internet connection for everything else, you can create a **split tunnel** client. For that, you just need to change the `AllowedIPs` field to your network subnet: `192.168.1.0/24`.

And you're done! From your smartphone you should be able to ping and access any device on your private network by their local IP. I want to thank to the many guides like [this](https://www.reddit.com/r/pihole/comments/bnihyz/guide_how_to_install_wireguard_on_a_raspberry_pi/) that I found online and helped me with this!