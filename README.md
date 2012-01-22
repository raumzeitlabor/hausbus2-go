# Initial version of an RZLBus library for Go.

See http://raumzeitlabor.de/wiki/Hausbus2 for the motivation and specification.

## Installing the library on your system

        (cd rzlbus && gomake && sudo gomake install)

## Running the example

First you need an SSL certificate; it’s sufficient to create a self-signed one
for now. See http://www.akadia.com/services/ssh_test_certificate.html

Then, compile and run the example code:

        gomake && ./example -rzlbus_listen="localhost:10444"
