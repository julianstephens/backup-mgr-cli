# Project Architecture

## Terminology

- pack: a collection of chunks
- chunk: byte data
- snapshot: the state of a directory backed up at some point in time

### Encryption/Decryption

![Crypto Workflow](./img/crypto-workflow.png)

- XChaCha20-POLY1305 authenticated encryption with associated data (AEAD)

  - XChaCha20: version of Google's ChaCha20 symmetric key encryption algorithm

    - generates a psuedo-random stream of bits (key stream) that's XOR'd with the plaintext to create random ciphertext
    - key stream is generated from encryption key and nonce
    - X version of ChaCha20 allows random nonce generation

  - POLY1305: message authenticated code (MAC) algorithm for validating encrypted data has not been modified

    - takes encrypted message, associated data, nonce, and secret key as inputs and produces MAC
    - on encryption, MAC is appended to the ciphertext
    - on decryption, MAC is derived and compared to the appended MAC to ensure zero modification

- Password derived keys
  - Argon2id: password hashing algorithm
  - 2 keys per user: file encryption key and key encryption key
    - file encryption key randomly generated and stored encrypted in ciphertext header
    - master encryption key derived from password and used to decrypt file encryption key
    - password change requires only decrypting and re-encrypting master key instead of all data

### File Chunking

- [FastCDC](https://www.usenix.org/system/files/conference/atc16/atc16-paper-xia.pdf) content-driven chunking strategy for data deduplication

  - optimizes hash for chunking by using increased zero padding to mimic Rabin-based CDC sliding window
  - enlarges minimum chunk sized for higher CDC speed
  - normalized chunking to reduce chunks with sizes at the poles

## Packing

### Pack Format

```
Blob1 | ... | BlobN | Header | KeyLength | HeaderLength
```

### Pack Header

```
headerData Blob1 | ... | headerData BlobN | encrypted session key |
```

```go
type headerData struct {
	Type               uint8
	Length             uint32
	UncompressedLength uint32
	ID                 warden.ID
}
```

#### Header Creation

1. create buf with capacity = # of blobs \* uncompressed blob data size
2. for each blob build header data and id and append to buf
3. encrypt buf
4. append encrypted buf len and encrypted session key to encrypted buf
5. verify header

#### Header Verification

1. read header data and length from end
2. ensure header length is non-zero and not too small or too large
3. read blobs from header data
4. ensure number of decoded blobs matches number of expected blobs
5. ensure decoded blob ids match expected ids

## Backups

1. get the lastest snapshot for backup path
2. for each file in backup path, if file has not been modified since last snapshot, then copy to new snapshot and go to next file. else continue
   1. for each chunk in the chunked file, calculate hash
      1. if the hash exists in a previous snapshot or in the backup medium, copy it's pack metadata and go to next chunk. else continue to step 2
      2. compress and encrypt chunk
      3. store chunk as blob or append to pack
      4. if pack size is >= min pack size, store pack and add metadata to new snapshot
3. chunk, compress, encrypt, and store new snapshot
