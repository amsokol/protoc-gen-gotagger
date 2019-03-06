@protoc --proto_path=./third_party --proto_path=./proto --proto_path=./test --go_out=./test data.proto

@protoc --proto_path=./third_party --proto_path=./proto --proto_path=./test --gotagger_out=xxx="bson+\"-\"",original_field_names=\"bson,graphql\",output_path=./test:./test data.proto
