The "FSM"-module
================
The finite state machine (fsm for short) handles how an elevator behaves with respect to local orders and the current configuration of the hardware. It alters the behavior of the system by calling functions from *elevio*.

Design choices
--------------
The FSM model consists of a few functions and a larger routine. The routine will start of by initializing the elevator, which generally means finding a floor to orient the system. It will then enter an infinite for-loop with a switch to the current elevator state, which is set to IDLE after the general initialization. Based on incoming orders, completed orders and potential errors, the fsm will change the elevator state, prompting a change of behavior through the switch statement. The module will notice changes to the orders through an internal go-routine, which reacts to updates on `orderToFsmChannel` where Order issues orders to the FSM. 

We made some simple choices to specify the behavior of the module. Firstly, the elevator only stops for orders in its current direction. This direction is set when the elevator receives an initial order from *IDLE* or when it changes the direction in *DOOR_OPEN*. Secondly, the door is set to stay open for `3 seconds`. Lastly, the elevator notices that the motor has stopped if it does not pass or reach a floor from *RUNNING* within `4 seconds`. The system will then retry to set the direction 3 times, before telling `Order` to reassign the elevators hall-orders to one of the online peers. 

Contents
--------
- **FsmShouldIStop:** A function checking if the elevator should stop at the current floor. 
- **FsmShouldIContinue** A function checking if the elevator should continue in its current direction. Checks if there are orders in that direction or if the elevator has reached its bounds. 
- **fsmOrdersAbove** A private function checking for orders to the elevator above our current floor. 
- **fsmOrdersBelow** A private function checking for orders to the elevator below our current floor. 
- **fsmOrdersAtMe** A private function checking for orders at our current floor. 
- **fsmInit** A private function to initialize the elevator. Drives it down one floor and turns on the floor indicator. 
- **FsmRoutine** A public function starting the FSM for the elevator. The function is called from main and runs as its own routine parallel to all the others. It runs a separate routine to check for incoming orders. The cases in the routine are changed through various events:
    - Case *IDLE*: The elevator is standing still, checking for orders at each iteration.
        - Event "*New order*": If a new order occurs, either above below or at the floor, the elevator will notice and go to *RUNNING* or *DOOR_OPEN* with the appropriate direction set.
        - Event "*Elevator not standing still*": Sets the motor direction to `MD_Stop`.
        - Event "*Motor stopped*": Elevator is sent to *IDLE* from *RUNNING* if power was cut from the motor. In *IDLE*, the elevator will stay unavailable till it re-initializes to a floor.
    - Case *RUNNING*: The elevator sets the motor direction from *IDLE* and starts a timer for which it wants to have reached the next floor. 
        - Event "*Sensor channel updated*": The elevator updates indicators and floor-variable before checking if it should stop at the floor. If it does, the state is set to *DOOR_OPEN*. If not it remains in *RUNNING*.
        - Event "*Error timer goes off*": If the elevator does not reach a floor in the required time, the fsm tries to reset the motor direction three times before failing. If this occurs, we tell `Order` that we have stopped and wish to reassign our orders to the other online peers. We further set ourselves to *IDLE*, into the **motor stopped**-event.
    - Case *DOOR_OPEN*: The elevator expedites an order at a floor and opens the door by turning on the door-open light for 3 seconds. It then generates an order with `order.Complete = True`, which is sent to `Order`.
        - Event "*fsmShouldIContinue*": The elevator sees more orders in the same direction. State is set to *RUNNING*. 
        - Event "*!fsmShouldIContinue*": The elevator sees no more orders in the same direction. State is set to *IDLE*. 
