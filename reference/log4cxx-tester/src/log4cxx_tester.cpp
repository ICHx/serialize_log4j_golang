#include <unistd.h>

#include <log4cxx/logger.h>
#include <log4cxx/mdc.h>

log4cxx::LoggerPtr logger(log4cxx::Logger::getLogger("Log4cxxTester"));

class Log4cxxTester {
public:
    void runTest();
};

void Log4cxxTester::runTest() {
    long idx = 0;
    while (true) {
        log4cxx::MDC::put("domainlog", "yes");
        log4cxx::MDC::put("originator", "Log4cxx-test");
        log4cxx::MDC::put("severity", "IMPORTANT");
        LOG4CXX_INFO(logger, "This is domain log message #" << idx << " from log4cxx.");
        log4cxx::MDC::remove("severity");
        log4cxx::MDC::remove("originator");
        log4cxx::MDC::remove("domainlog");

        LOG4CXX_INFO(logger, "This is debug message #" << idx << " from log4cxx.");

        usleep(2000000);
        idx++;
    }
}

int main() {
    char hostname[254];
    log4cxx::MDC::put("application", "Log4cxx-tester");
    if (gethostname(hostname, sizeof hostname) == 0) {
        log4cxx::MDC::put("host", hostname);
    }

    Log4cxxTester tester;
    tester.runTest();

    return 0;
}
