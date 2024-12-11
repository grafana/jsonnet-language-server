(import 'basic-object.jsonnet') +
{
  aaa: 'hello',
} + {
  bar: self.foo,
}
