{
  test: 'test',
  sub: {
    test2: self.test,  // Should not be found
  },

  sub2: {
    test3: 'test3',
  },

  sub3: self.sub2 {  // sub2 should be found
    test4: self.test3,  // test3 should be found
  },
}
