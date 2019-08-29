TODO:

* Handle IsRepeated on fields. These should generate a std::vector<T> for the field.
  Repeated instances when decoding should append to that vector.
  Repeated instances when encoding should be pushed to the stream.
  We also need to handle packed fields:
      ... Otherwise, all of the elements of the field are packed into a single
      key-value pair with wire type 2 (length-delimited). Each element is encoded
      the same way it would be normally, except without a key preceding it.
