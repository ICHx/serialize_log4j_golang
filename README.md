# serialize_log4j_golang

This project aim to receive log4jEvent from Log4j1.x socketAppender, and then convert it as json, to handle logs generated by legacy Java apps.

## Split the stream

The problem with my current hacky approach to split the stream is, it can sometimes successfully parsed by sop, while sometimes not.

# TODO:

- [x] close connection on forwarding finished? shouldn't
- [ ] retry client connection on failure?
- [x] LEVEL not working, ref: https://github.com/qos-ch/reload4j/blob/master/src/main/java/org/apache/log4j/Priority.java
- [ ] log4cxx not supported
