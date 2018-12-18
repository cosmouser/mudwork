package data
  
func Diff(s1, s2 []string) []string {
        result := []string{}
        for _, i := range s1 {
                found := false
                for _, j := range s2 {
                        if i == j {
                                found = true
                        }
                }
                if found == false {
                        result = append(result, i)
                }
        }
        return result
}
