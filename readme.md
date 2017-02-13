# Raft

Raft is a distrubuted consensus algorithm.

> Raft is a consensus algorithm that is designed to be easy to understand. It's equivalent to Paxos in fault-tolerance and performance. The difference is that it's decomposed into relatively independent subproblems, and it cleanly addresses all major pieces needed for practical systems. We hope Raft will make consensus available to a wider audience, and that this wider audience will be able to develop a variety of higher quality consensus-based systems than are available today.

From: http://raft.github.io

## Design

This raft implementation uses protocol buffers to communicate between nodes. The raft implementation is basic but works. It will not survive chaos scenarios as the configuration change section of the paper is not implemented. Only the basics are implemented here.
