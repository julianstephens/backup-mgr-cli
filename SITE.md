# Warden — Encrypted Backup CLI

> Note: This project is still in progress. Interfaces and implementation details may change as development continues.

Warden is a command-line tool for creating secure, end-to-end encrypted backups. It aims to be a simple, reliable, and cross-platform way to protect your data with strong cryptography, minimal configuration, and an approachable workflow.

## Goals

- End-to-end encryption by default
- Simple CLI with sensible defaults
- Fast, streaming-friendly backups
- Snapshot-based restore with integrity verification
- Portable single binary with minimal dependencies

## Concepts

- Repository: Logical destination that stores encrypted backup data and metadata.
- Snapshot: A point-in-time capture of input paths and metadata, used for restore.
- Target: Where backup data is persisted (e.g., local filesystem; remote options planned).
- Key Material: Credentials required to encrypt/decrypt data; must be kept secret and backed up.

## Security Model

Warden is designed so that:

- Encryption happens client-side; plaintext never leaves your machine.
- Integrity is authenticated; tampering is detectable.
- Key material is never written to the repository in a recoverable form.

Implementation specifics (algorithms, KDFs, key storage) will be documented as they solidify. Until then, treat outputs and formats as experimental.

## Storage Backends

- Local filesystem repository (MVP)
- Planned: Remote object storage (e.g., S3-compatible), SSH/SFTP, and others

## Features

Current (MVP targets):

- Encrypted snapshot creation
- Listing and restoring snapshots
- Repository initialization and health checks

Planned:

- Incremental backups and content addressing
- Compression and/or deduplication
- Repository verification and periodic integrity scans
- Native support for common cloud/object storage
- Key rotation and backup key escrow options
