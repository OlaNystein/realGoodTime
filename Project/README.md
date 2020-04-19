TTK4145 - Real Time Programming, elevator project
=================================================
The goal of this project was to design a system for `n` elevators to work in parallel over `m` floors using a selection of techniques to support the chosen language and strategy of implementation. Our group decided on using the ever more popular language `Google GO` , which bases itself on message passing between internal processes instead of synchronizing a set of shared variables. We also decided on our elevators working in a 'peer-to-peer' fashion, meaning that every elevator have information about the current state, direction and floor of all the peers on the network and can distribute orders freely. When an elevator then receives an order, it calculates which elevator is best to handle the given order and informs the other over a `UDP broadcast`. 

In this repository one can find our software for solving the described problem. `main.go` runs the software for a single elevator using the routines specified in other modules, who are located in their own respective folders. For detailed information about said modules and their routines, reference the `README` files of each such folder. This README file will contain the specifications for which we based our implementation upon, as well as a more thorough explanation of our design and the performance of the system. 

Problem description
-------------------
The project set a few requirements for the system which we carefully monitored during implementation. For the full list of requirements, see [`the problem description for TTK4145`](https://github.com/TTK4145/Project/blob/master/README.md). This list, which is fairly extensive, can be boiled down to the following: 
- **No orders are lost:** When an order has been accepted and a lighthas turned on, the order should be expedited by one of the elevators. The system should then be robust to both packet-loss over the broadcast and elevators disconnecting or crashing. 
- **Multiple elevators are better than one:** The use of multiple elevators should be sensible, and yield a better performance through co-operation.
- **Individual elevators behave sensibly:** Each elevator behaves logically in it's own right, not stopping at every floor and reacting differently to different orders. 
- **Lights work as expected:** When an order is taken or handled, the appropriate ligths toggle as expected. The floor-indicator also works as expected. 

Model assumptions
-----------------
During implementation, we were also allowed to make certain assumptions. The most prevalent ones being that only one error would occur at a time and that one elevator would always work as expected. Again, reference [`the problem description`](https://github.com/TTK4145/Project/blob/master/README.md) for the full list of assumptions. 

Design and performance
----------------------
As stated above, our implementation was of a `peer-to-peer` network using message-passing and UDP-broadcasts. We chose to do it this way for a few reasons. Mainly, we found peer-to-peer easier to handle than a master-slave system, which we initially opted for when designing the software. If every elevator knew everything about every other elevator at all times, handling loss of connection, reconnecting and delegation of orders became significantly easier. All one needed to do was let each elevator designate the orders it received to one of the peers, reassign the orders of a lost elevator to the others and update elevators that entered the network with their previous orders or the state of all connected elevators. This was done quite easily by letting each elevator update a large local list with the orders, states, directions and floors of all peers. When a change occurred at an elevator it broadcasts this list to the other elevators, prompting them to update their own. This resulted in a lot of information being passed over the network, but it worked very well. 

The choice of using UDP connections over TCP was simply a matter of ease. UDP, though more prone to packet-loss than TCP, is both easier to set up and handle on such a local system. By using a simple static redundancy sending for each order, which is not the prettiest method due to the sometimes unnecessary resource-usage on the network, we were then able to establish simple, robust communication between all of the elevators. Finally, the choice of using message passing over shared-variable synchronization was simply related to using the GO programming language. We wished to attempt using it mainly for its surge of popularity in practice and relevancy towards future projects. 

Overall, the final implementation worked out great! The elevators worked well together, and all requirements for functionality, order handling, lights and behavior were met. The amount of information passed between systems is definitely large, but it didn't seem to cause any problems over broadcasts or between processes. 

How to run the software
-----------------------
