from ctypes import *


class _GoString(Structure):
    _fields_ = [
        ("p", c_char_p),
        ("n", c_longlong)
    ]


def GoString(s):
    if not isinstance(s, (str, unicode)):
        raise TypeError("invalid type: %s" % str(type(s)))
    x = str(s)
    return _GoString(x, len(x))


def GoBinding(library, *argTypes):
    """
    Get a binding to a go function of the same name in the given .so
    The python function itself is not run, and can return or pass

    :param argTypes: *parameter types
    """

    lib = cdll.LoadLibrary(library)

    def inner(fn):
        _types = {
            str: _GoString
        }

        _getters = {
            str: GoString
        }

        for v in argTypes:
            if not isinstance(v, type):
                raise TypeError("invalid argument type")
            if v not in _types:
                raise TypeError("type does not have a mapping: %s" % str(type(v)))

        _go_func = getattr(lib, fn.__name__)
        _go_func.argtypes = [_types[v] for v in argTypes]

        def _wrap(*args):
            _args = [_getters[arg_type](arg) for arg_type, arg in zip(argTypes, args)]
            return _go_func(*tuple(_args))
        return _wrap
    return inner


@GoBinding("./lbryschema-python-binding.so", str, str, str, str)
def VerifySignature(claim, certificate, claim_address, certificate_id):
    pass


cert_claim_hex = "08011002225e0801100322583056301006072a8648ce3d020106052b8104000a03420004d015365a40f3e5c03c87227168e5851f44659837bcf6a3398ae633bc37d04ee19baeb26dc888003bd728146dbea39f5344bf8c52cedaf1a3a1623a0166f4a367"
signed_claim_hex = "080110011ad7010801128f01080410011a0c47616d65206f66206c696665221047616d65206f66206c696665206769662a0b4a6f686e20436f6e776179322e437265617469766520436f6d6d6f6e73204174747269627574696f6e20342e3020496e7465726e6174696f6e616c38004224080110011a195569c917f18bf5d2d67f1346aa467b218ba90cdbf2795676da250000803f4a0052005a001a41080110011a30b6adf6e2a62950407ea9fb045a96127b67d39088678d2f738c359894c88d95698075ee6203533d3c204330713aa7acaf2209696d6167652f6769662a5c080110031a40c73fe1be4f1743c2996102eec6ce0509e03744ab940c97d19ddb3b25596206367ab1a3d2583b16c04d2717eeb983ae8f84fee2a46621ffa5c4726b30174c6ff82214251305ca93d4dbedb50dceb282ebcb7b07b7ac65"
claim_addr = "bSkUov7HMWpYBiXackDwRnR5ishhGHvtJt"
cert_id = "251305ca93d4dbedb50dceb282ebcb7b07b7ac65"
import time
from lbryschema.decode import smart_decode

cd = smart_decode(signed_claim_hex)
certd = smart_decode(cert_claim_hex)


def clock_lbryschema_python(n=10.0):
    start = time.time()
    for i in range(int(n)):
        assert cd.validate_signature(claim_addr, certd)
        if i % 10 == 0:
            print i
    avg = float(time.time() - start) / n
    return 1.0 / avg


def clock_lbryschema_go(n=100.0):
    start = time.time()
    for i in range(int(n)):
        assert VerifySignature(signed_claim_hex, cert_claim_hex, claim_addr, cert_id)
        if i % 10 == 0:
            print i
    avg = float(time.time() - start) / n
    return 1.0 / avg

print "Start"
print "%f validations / second with python" % clock_lbryschema_python()
print "%f validations / second with go binding" % clock_lbryschema_go()

