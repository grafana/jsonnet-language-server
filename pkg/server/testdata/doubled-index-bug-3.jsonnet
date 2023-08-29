local g = import 'doubled-index-bug-2.jsonnet';
{
  // completing fields of `g.hello` should get use `g.hello.to`, not `g.hello.hello`
  a: g.hello,
}
