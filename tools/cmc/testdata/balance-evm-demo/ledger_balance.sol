pragma solidity ^0.4.11;

contract LedgerBalance {
    mapping(address => uint) public balances;

    function updateMyBalance(uint newBalance) public {
        balances[msg.sender] = newBalance;
    }
    function updateBalance(uint _newBalance, address _to) public {
        balances[_to] = _newBalance;
    }

    function transfer(address _to, uint256 _value) public  returns (bool success) {
        require(balances[msg.sender] >= _value);
        require(balances[_to] + _value >= balances[_to]);
        balances[msg.sender] -= _value;
        balances[_to] += _value;
        emit Transfer(msg.sender, _to, _value);
        return true;
    }
    event Transfer(address indexed _from, address indexed _to, uint256 _value);
}