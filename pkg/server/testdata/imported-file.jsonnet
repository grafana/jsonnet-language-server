local otherfile = import 'basic-object.jsonnet';

{
  a: otherfile.foo,
  b: otherfile.bar,
}
