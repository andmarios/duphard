# duphard — duplicate 2 hard link

## Usage

`duphard` is a small utility that detects duplicate files and replaces them with hard links.
It is pronounced dup-hard. It has been tested on GNU/Linux and ext4 filesystem.

Duphard is a bit naive and expects you to provide it with files and directories in the
same filesystem since only then hard links can work. If you fail to do so, then if you run
`duphard` in non dry-run mode, it will delete one copy of your first duplicate file and exit
with an error code.

It is easy to run it. In dry-run (no changes made):

    duphard <DIR> [<FILE> <DIR>...]

In this mode it will search recursively for regular files in all paths provided and report to you
how many and which are duplicates and an estimate of the space you can save by converting them to hard links.

If you are convinced about the results, you can run it in non-dry mode. This will delete duplicate
files and replace them with hard links.

    duphard -d=0 <DIR> [<FILE> <DIR>...]

__Please remember__, hard links are still links, they point to the same inode (data) in the filesystem.
Changes to one file will reflect to others sharing the same data. Thus you should use hard links on files
that for all purposes are immutable (e.g your media collection) or you really expect to have the same
content (e.g a banner across different website directories).

## Behind the scenes

Duphard starts by grouping your files by size. Files of the same size go to the same group. Groups
with more than one file are obviously duplicate candidates.

Duphard will check if there are already any hard links in each group since it is an easy and quick check
and remove them from the list.

Then, for each group with more than one members, it calculates a checksum (md5) for each file and creates
a new map where files are grouped by checksum.

This list contains duplicate files and is reported to you. In non dry-run mode, each duplicate file gets
deleted first, then hardlinked. If any error occurs while deleting/hardlinking, duphard will immediately
stop in order to prevent further errors.
