cd "%~dp0..\pkg\channeldpb"
protoc --go_out=. --go_opt=paths=source_relative -I . *.proto

cd "%~dp0..\kooola\chatpb"
protoc --go_out=. --go_opt=paths=source_relative -I . *.proto
