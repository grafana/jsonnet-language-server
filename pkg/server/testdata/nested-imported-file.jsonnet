local file = (import 'import-intermediary.libsonnet').otherFile;
{
  foo: file.foo,
}
