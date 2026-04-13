#!/bin/bash
# Enable IP forwarding
echo "net.ipv4.ip_forward=1" > /etc/sysctl.d/nat.conf
sysctl -p /etc/sysctl.d/nat.conf

# Install iptables (iptables-nft already present, just need iptables-services for persistence)
dnf install -y iptables-services

# Detect primary interface (ens5 on AWS Nitro)
IFACE=$(ip route show default | awk '/default/ {print $5}' | head -1)

# NAT masquerading
iptables -t nat -A POSTROUTING -o "$IFACE" -j MASQUERADE
iptables -A FORWARD -j ACCEPT

# Persist rules across reboots
systemctl enable iptables
service iptables save
