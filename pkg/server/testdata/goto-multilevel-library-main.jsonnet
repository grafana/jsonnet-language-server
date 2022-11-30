local library = import 'goto-multilevel-library-top.libsonnet';
{
  my_item: library.sub1.subsub1.new(name='test'),
}
