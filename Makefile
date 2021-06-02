build:
	go build beam.go

run:
	./beam

spark-run:	build
	./beam --runner PortableRunner --endpoint localhost:8099
	
