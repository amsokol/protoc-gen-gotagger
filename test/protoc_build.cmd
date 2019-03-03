@protoc --proto_path=. --go_out=. data.proto

@protoc --proto_path=. --gotagger_out=xxx="bson+\"-\"",output_path=.:. data.proto
