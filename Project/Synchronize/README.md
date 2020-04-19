The "Synchronize"-module
========================
The synchronizer module is tasked with, as the name suggests, synchronizing data between the peers on the network. This translates into keeping an eye on what elevators are currently online on the network, receiving and sending orders and updates to said elevators and synchronizing the information between the local `Order`-module and the online peers. Basically, every elevator has a synchronize-module to keep communication flowing between all the elevators on the network.

Design choices
--------------
As the synchronizer handles all communication over the network, it was a natural decision to let this be the only other module outside of `main` using functions from the `network`-module. This further made it natual to let the synchronizer keep track of the other elevators on the network and whether they were online or not. This was implemented through a second routine, `ConnectedElevatorsRoutine`, which is called from `main` alongside `SynchronizerRoutine`.

Related to the fact that this module is the only one receiving orders from other elevators on the network, we found it natural to let the synchronizer handle updates to the local `elevatorList` variable. Upon receiving orders from either a peer on the network or the local `Order`-module, synchronize will update this list with the recorded changes before either passing it to `Order` or broadcast it to the other peers. 

One important choice we made in this module, was the choice of using a static-redundancy method of broadcasting packages from a peer to the others. This basically means that when sending a packet over the network, the elevator sends multiple copies of the package rather than wait for acknowledgements to a single sending. This is more resource-intensive, but it's implementation is a lot prettier and is naturally resistant to packet loss. For example: If 1/4 of packets were persumed lost over the network, the probability of the other elevators not getting the packet would be significantly lower if it was sent 5-10 times. 

Contents
--------
- **syncClearNonLocalOrders:** A private function used to clear orders assigned to other elevators on the network from `elevatorList`. Used in case of timing out on the network. 
- **checkOnlineElevators:** A private function to check how many elevators are currently online on the network. 
- **ConnectedElevatorsRoutine:** A public function called from main. Using the peers-logic from `network` it updates the `onlineElevators` list used by both the `synchronize` and `Order` modules. It will also keep track of if our local elevator has timed out, upon which it updates `timedOutChannel`.
- **SynchronizerRoutine:** A publiv function called from main. This routine handles the communication of the elevator, keeping local data up to date with data of the online elevators. Upon initalization it will check if there are other elevators online. If there are, it will receive the current list of all elevators on the network, including order assigned to itself, from all the peers online. The routine also starts a subroutine checking for updates to the online elevators. This is naturally where online elevators will notice new elevators on the network and send them the information about the current system. The routine also reacts to changes to our own network status through updates to `timedOutChannel`. After initializing, the routine will enter an infinite for-loop and respond to the following cases: 
    - Case "*Update to the local elevator*": Updates everything but the local order-queue. Sends the elevator list back to `Order`. 
    - Case "*New order recorded*": `Order` sends a new order from the hardware. Depending on the nature of the order and the elevators online-status, different outcomes can occur. Regardless of the outcome, the routine will create a `config.Message` containing all changes and broadcast it to the online peers. The possible outcomes are:
        - Status '*Order is assigned to me and is complete*': Set the corresponding element in the local elevator list to `False`.
        - Status '*Order is assigned to me and is not complete*': Set the corresponding element in the local elevator list to `True`. 
        - Status '*Order is assigned to another elevator and is completed*': Set the corresponding element for the relevant elevator in the local elevator list to `False`. This occurs upon orders being reassigned. 
        - Status '*Order is assigned to another elevator, is not complete and that elevator is online*': Set the corresponding element for the relevant elevator in the local elevator list to `True`.
        - Status '*Order is assigned to another elevator, is not complete and that elevator is offline*': No update happens. Cannot record an order to an offline elevator.  
    - Case "*Synchronize receives a broadcast from another peer*": Upon receiving a `config.Message` from one of the online peers, the elevator will update its information about all elevators but its own. It will then check for an order attached to the message. If the `config.Message.NewOrder.ID` equals `myID`, the order will be recorded for the local elevator. 

