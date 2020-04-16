The "FSM"-module (WIP)
======================
The finite state machine (fsm for short) handles how an elevator behaves with respect to local orders and the current configuration of the hardware. It alters the behavior of the system by calling functions from *elevio*.

Design choices
--------------
The FSM model consists of a few functions and a larger routine. The routine will start of by initializing the elevator, which generally means finding a floor to orient the system. It will then enter an infinite for-loop with a switch to the current elevatorstate, which is set to IDLE after the general initialization. Based on incoming orders, completed orders and potential errors, the fsm will change the elevator state, prompting a change of behavior through the switch statement. The module will notice changes to the orders through an internal go-routine, which reacts to updates on `orderToFsmChannel` where control issues orders to the FSM. 

We made some simple choices to specify the behavior of the module. Firstly, the elevator only stops for orders in its current direction. This direction is set when the elevator receives an initial order from *IDLE* or when it changes the direction in *DOOR_OPEN*. Secondly, the door is set to stay open for `3 seconds`. Lastly, the elevator notices that the motor has stopped if it does not pass or reach a floor from *RUNNING* within `4 seconds`. The system will then retry to set the direction 3 times, before telling *Control* to reassign the elevators hall-orders to one of the online peers. 

Contents
--------
 - **FsmShouldIStop:** A function checking if the elevator should stop at a current floor. If an elevator is 