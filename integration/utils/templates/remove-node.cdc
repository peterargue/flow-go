import FlowIDTableStaking from 0xIDENTITYTABLEADDRESS

// This transaction removes an existing node from the identity table
// by unstaking it and removing it from the approved list
transaction(id: String) {

    // Local variable for a reference to the ID Table Admin object
    let adminRef: &FlowIDTableStaking.Admin

    prepare(acct: AuthAccount) {
        // borrow a reference to the admin object
        self.adminRef = acct.borrow<&FlowIDTableStaking.Admin>(from: FlowIDTableStaking.StakingAdminStoragePath)
            ?? panic("Could not borrow reference to staking admin")
    }

    execute {
        self.adminRef.removeAndRefundNodeRecord(id)
        let nodeIDs = FlowIDTableStaking.getApprovedList()
       	nodeIDs[id] = nil

        // set the approved list to the new allow-list
        self.adminRef.setApprovedList(nodeIDs)
    }
}


