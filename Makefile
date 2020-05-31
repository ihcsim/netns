NETWORK_NAMESPACE ?= firefly

VETH_LINK_PEER_0 ?= veth0
VETH_LINK_PEER_1 ?= veth1

define netns
	sudo ip netns ${1} "${NAME}"
endef

netns-%:
	echo "$*"
	$(call netns,$*)

link-veth:
	# see https://www.man7.org/linux/man-pages/man4/veth.4.html
	sudo ip link add "${VETH_LINK_PEER_0}" type veth peer name "${VETH_LINK_PEER_1}"
	sudo ip link set "${VETH_LINK_PEER_1}" netns "${NETWORK_NAMESPACE"}
