# Package Paladin

Package paladin provides simple protection of critical resources against
asynchronous interruption signals sent by the operating systems.
Paladin provides a Run method that expects 

- a function to obtain a resource (which must be a Closer)

- a function to release the resource (using Close)

- and a function that is run in between
obtaining and releasing the resource; the user application
should entirely live within this function.

Currently, only SIGINT is handled and the behaviour is to
close the program.
More sophisticated behaviour and more signals will be provided
in the future.

