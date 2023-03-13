# log4jtest

copy log4j.jar into current dir
make
make test

will connect to localhost:2518 and send a logging message

To capture: nc -l 2518 > log4j.capture

TODO:

- [x] close connection on forwarding finished? shouldn't
- [ ] retry client connection on failure?
- [x] LEVEL not working, ref: https://github.com/qos-ch/reload4j/blob/master/src/main/java/org/apache/log4j/Priority.java
