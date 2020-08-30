using Go = import "/go.capnp";
@0xa99aba6cb134a1c0;
$Go.package("captn");
$Go.import("captn/request");

struct Request {
    method @0 :Data;
    url @1 :Data;
    body @2 :Data;
    timestamp @3 :Data;
}

struct RequestGroup {
  requests @0 :List(Request);
}
