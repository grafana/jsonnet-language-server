local otherfile = import 'goto-basic-object.jsonnet';

{
  a: otherfile.foo,
  b: otherfile.bar,
}
