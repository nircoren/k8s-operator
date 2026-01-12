package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto copies the receiver into out
func (in *Greeting) DeepCopyInto(out *Greeting) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy creates a deep copy of Greeting
func (in *Greeting) DeepCopy() *Greeting {
	if in == nil {
		return nil
	}
	out := new(Greeting)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object
func (in *Greeting) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies the receiver into out
func (in *GreetingList) DeepCopyInto(out *GreetingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Greeting, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a deep copy of GreetingList
func (in *GreetingList) DeepCopy() *GreetingList {
	if in == nil {
		return nil
	}
	out := new(GreetingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object
func (in *GreetingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies GreetingSpec
func (in *GreetingSpec) DeepCopyInto(out *GreetingSpec) {
	*out = *in
}

// DeepCopyInto copies GreetingStatus
func (in *GreetingStatus) DeepCopyInto(out *GreetingStatus) {
	*out = *in
}
