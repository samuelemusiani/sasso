<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>
        Network interfaces allow your VM to communicate using the network. You can have from 0 up to
        32 interfaces, but 0 is not recommended because it means that your VM will not be able to
        communicate and you will not be able to access it.
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Network</template>
      <p>
        The first thing you have to choose when creating a network interface is the network. The
        networks from which you can choose are the ones that you have created or the ones shared by
        your group. You can find all the networks in the
        <RouterLink to="/net" class="link link-primary">networks page</RouterLink>.
      </p>
      <p>
        You have to think of a network as a switch, and every network is isolated from the others.
        Interfaces form different VMs will be able to communicate with each other only if they are
        connected to the same network. You can allow inter-network communication by creating a VM
        with at least two interfaces connected to different networks and configuring it as a router.
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>IP addess and Gateway</template>
      <p>
        When you have chosen the network, you will see the corresponding subnet, gateway and
        broadcast address. In the IP address field you can choose the IP address of the interface.
        You can choose any IP address, but you will only be able to access internet from the VM or
        the VM itself if you choose an IP address that in network's subnet and it's not the same as
        the network, gateway or the broadcast address.
      </p>
      <p>
        You can only have one interface with the gateway IP address. The gateway allows
        communication outside the network. If you want to reach internet from your VM or to reach
        your VM, you need to put the network's gateway IP address.
      </p>
    </HelpParagraph>
    <HelpParagraph>
      <template #title>VLANs</template>
      <p>
        If you choose a network that supports VLANs, you can add a tag to the interface. Inside the
        VM you will see untagged traffic, but as soon as the traffic goes out of the VM, it will be
        tagged with the VLAN tag you have chosen. VMs on the same network with different VLAN tags
        will not be able to communicate with each other. To communicate with the gateway you have to
        use VLAN tag 0.
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
