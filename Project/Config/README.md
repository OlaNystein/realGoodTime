The "Config"-module
===================
The configuration-module declares all datastructures used in and between other modules on an elevator. 

Contents
--------
  - Constants: 
    - **NumElevators:** The number of elevators in the system.
    - **NumFloors:** The number of floors for each elevator.
    - **NumButtonTypes:** The number of button-types for each elevator. The elevator handles 3 at most, for `UP`, `DOWN` and `CAB` orders.
    - **_initialized:** ??
    - **_mtx:** ??
    - **_conn** ??
  - Hardware datastructures: 
    - **Pollrate:** A variable used by elevio to set the frequency of polling-routines.
    - **ButtonType:** Enumerators for the various buttons.
    - **MotorDirection:** Enumerators for the three motordirections `Up`, `Down` and `Stop`.
    - **ButtonEvent:** Struct to organize information from PollButtons in elevio. Contains when button was pressed at what floor. 
  - Communication datastructures:
    - **ElevState:** Enumerators for the possible states of an elevator.
    - **Elev:** A struct organizing all the information about an elevator.
    - **Order:** A struct defining an order. An elevator `ID` receives an order defined by `Button` at floor `Floor`. `Complete` tells if the order is expedited or not. 
    - **Message:** A struct defining a message to be sent between elevators. It includes the elevators updated elevatorlist `ElevList` as well as an updated order `Order` and the sender `ID`.
  

