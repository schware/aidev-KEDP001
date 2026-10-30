// Package kedpv1은 b-proto/kedp/v1/processor.proto(Go–Rust 약속서)로 만든 gRPC 코드다.
// processor.pb.go, processor_grpc.pb.go는 아래 명령이 만든다. 손으로 고치지 않는다.
// proto가 바뀌면 b-go-end 폴더에서 go generate ./internal/kedpv1 을 다시 실행한다.
package kedpv1

//go:generate protoc -I ../../../b-proto --go_out=../.. --go_opt=module=github.com/schware/aidev-KEDP001/b-go-end --go_opt=Mkedp/v1/processor.proto=github.com/schware/aidev-KEDP001/b-go-end/internal/kedpv1 --go-grpc_out=../.. --go-grpc_opt=module=github.com/schware/aidev-KEDP001/b-go-end --go-grpc_opt=Mkedp/v1/processor.proto=github.com/schware/aidev-KEDP001/b-go-end/internal/kedpv1 kedp/v1/processor.proto
