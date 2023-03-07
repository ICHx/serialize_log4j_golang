all: output/log4cxx-tester output/log4cxx.properties

output/log4cxx-tester: src/log4cxx_tester.cpp
	g++ -o $@ $< -Wl,-Bstatic -llog4cxx -laprutil-1 -lapr-1 -Wl,-Bdynamic -lpthread -lexpat 

output/log4cxx.properties: log4cxx.properties
	cp $< $@

clean:
	rm -f output
